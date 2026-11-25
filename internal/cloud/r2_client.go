package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	"t-deck/internal/config"
)

// ErrObjectNotFound indicates the requested object key does not exist in storage.
var ErrObjectNotFound = errors.New("object not found")

// ObjectStore abstracts cloud object storage operations for R2 and unit tests.
type ObjectStore interface {
	UploadFile(ctx context.Context, key string, localPath string, contentType string) error
	DownloadFile(ctx context.Context, key string, localPath string) error
	DeleteObject(ctx context.Context, key string) error
	PresignGetURL(ctx context.Context, key string, expiry time.Duration) (string, error)
	PutJSON(ctx context.Context, key string, payload any) error
	GetJSON(ctx context.Context, key string, out any) (bool, error)
	ObjectExists(ctx context.Context, key string) (bool, error)
	TestConnection(ctx context.Context) error
}

// R2Client implements ObjectStore backed by Cloudflare R2 via aws-sdk-go-v2 S3 API.
type R2Client struct {
	client *s3.Client
	bucket string
}

// NewR2Client validates R2Config and constructs an S3-compatible client targeting Cloudflare R2.
func NewR2Client(ctx context.Context, cfg config.R2Config) (*R2Client, error) {
	if !cfg.Enabled {
		return nil, errors.New("R2 sync is disabled in settings")
	}
	if strings.TrimSpace(cfg.AccountID) == "" ||
		strings.TrimSpace(cfg.AccessKeyID) == "" ||
		strings.TrimSpace(cfg.SecretAccessKey) == "" ||
		strings.TrimSpace(cfg.BucketName) == "" {
		return nil, errors.New("missing required Cloudflare R2 credentials or bucket name")
	}

	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", strings.TrimSpace(cfg.AccountID))
	awsCfg, err := awsconfig.LoadDefaultConfig(
		ctx,
		awsconfig.WithRegion("auto"),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("init R2 aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})

	return &R2Client{
		client: client,
		bucket: strings.TrimSpace(cfg.BucketName),
	}, nil
}

// TestConnection verifies R2 credentials and bucket access.
func (c *R2Client) TestConnection(ctx context.Context) error {
	_, err := c.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(c.bucket),
		MaxKeys: aws.Int32(1),
	})
	if err != nil {
		return fmt.Errorf("R2 connection failed: %w", err)
	}
	return nil
}

// UploadFile uploads a local file to R2 under key.
func (c *R2Client) UploadFile(ctx context.Context, key string, localPath string, contentType string) error {
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open %s: %w", localPath, err)
	}
	defer f.Close()

	input := &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
		Body:   f,
	}
	if strings.TrimSpace(contentType) != "" {
		input.ContentType = aws.String(contentType)
	}

	if _, err := c.client.PutObject(ctx, input); err != nil {
		return fmt.Errorf("upload %s: %w", key, err)
	}
	return nil
}

const (
	multipartThresholdBytes = 15 * 1024 * 1024 // 15 MB
	multipartChunkBytes     = 10 * 1024 * 1024 // 10 MB
	multipartConcurrency    = 4
)

type progressReader struct {
	r          io.Reader
	total      int64
	done       atomic.Int64
	onProgress func(done, total int64)
}

func (p *progressReader) Read(buf []byte) (int, error) {
	n, err := p.r.Read(buf)
	if n > 0 && p.onProgress != nil {
		cur := p.done.Add(int64(n))
		p.onProgress(cur, p.total)
	}
	return n, err
}

// UploadFileWithProgress uploads localPath to R2 using PutObject for files < 15 MB
// and concurrent S3 Multipart Upload (10 MB parts, concurrency 4) for larger artifacts.
func (c *R2Client) UploadFileWithProgress(ctx context.Context, key string, localPath string, contentType string, onProgress func(done, total int64)) error {
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open %s: %w", localPath, err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat %s: %w", localPath, err)
	}
	totalSize := info.Size()

	if totalSize < multipartThresholdBytes {
		pr := &progressReader{r: f, total: totalSize, onProgress: onProgress}
		input := &s3.PutObjectInput{
			Bucket:        aws.String(c.bucket),
			Key:           aws.String(key),
			Body:          pr,
			ContentLength: aws.Int64(totalSize),
		}
		if strings.TrimSpace(contentType) != "" {
			input.ContentType = aws.String(contentType)
		}
		if _, err := c.client.PutObject(ctx, input); err != nil {
			return fmt.Errorf("upload %s: %w", key, err)
		}
		return nil
	}

	createIn := &s3.CreateMultipartUploadInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}
	if strings.TrimSpace(contentType) != "" {
		createIn.ContentType = aws.String(contentType)
	}
	createOut, err := c.client.CreateMultipartUpload(ctx, createIn)
	if err != nil {
		return fmt.Errorf("create multipart upload %s: %w", key, err)
	}
	uploadID := aws.ToString(createOut.UploadId)

	abortOnErr := func() {
		_, _ = c.client.AbortMultipartUpload(context.Background(), &s3.AbortMultipartUploadInput{
			Bucket:   aws.String(c.bucket),
			Key:      aws.String(key),
			UploadId: aws.String(uploadID),
		})
	}

	numParts := int((totalSize + multipartChunkBytes - 1) / multipartChunkBytes)
	type partResult struct {
		num  int32
		etag string
	}
	results := make([]partResult, numParts)
	sem := make(chan struct{}, multipartConcurrency)
	var wg sync.WaitGroup
	var errOnce sync.Once
	var firstErr error
	var uploadedBytes atomic.Int64

	for partIdx := 0; partIdx < numParts; partIdx++ {
		partIdx := partIdx
		offset := int64(partIdx) * multipartChunkBytes
		partLen := multipartChunkBytes
		if offset+int64(partLen) > totalSize {
			partLen = int(totalSize - offset)
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			buf := make([]byte, partLen)
			n, readErr := f.ReadAt(buf, offset)
			if readErr != nil && readErr != io.EOF {
				errOnce.Do(func() { firstErr = fmt.Errorf("read part %d: %w", partIdx+1, readErr) })
				return
			}
			buf = buf[:n]

			partNum := int32(partIdx + 1)
			out, upErr := c.client.UploadPart(ctx, &s3.UploadPartInput{
				Bucket:        aws.String(c.bucket),
				Key:           aws.String(key),
				UploadId:      aws.String(uploadID),
				PartNumber:    aws.Int32(partNum),
				Body:          bytes.NewReader(buf),
				ContentLength: aws.Int64(int64(len(buf))),
			})
			if upErr != nil {
				errOnce.Do(func() { firstErr = fmt.Errorf("upload part %d: %w", partNum, upErr) })
				return
			}
			results[partIdx] = partResult{num: partNum, etag: aws.ToString(out.ETag)}
			done := uploadedBytes.Add(int64(len(buf)))
			if onProgress != nil {
				onProgress(done, totalSize)
			}
		}()
	}
	wg.Wait()

	if firstErr != nil {
		abortOnErr()
		return firstErr
	}

	sort.Slice(results, func(i, j int) bool { return results[i].num < results[j].num })
	completedParts := make([]types.CompletedPart, len(results))
	for i, r := range results {
		completedParts[i] = types.CompletedPart{
			PartNumber: aws.Int32(r.num),
			ETag:       aws.String(r.etag),
		}
	}

	if _, err := c.client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(c.bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: completedParts,
		},
	}); err != nil {
		abortOnErr()
		return fmt.Errorf("complete multipart upload %s: %w", key, err)
	}
	return nil
}

// DownloadFile downloads an object from R2 to localPath atomically via temp file.
func (c *R2Client) DownloadFile(ctx context.Context, key string, localPath string) error {
	out, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFoundErr(err) {
			return fmt.Errorf("%w: %s", ErrObjectNotFound, key)
		}
		return fmt.Errorf("download %s: %w", key, err)
	}
	defer out.Body.Close()

	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return fmt.Errorf("create dir for %s: %w", localPath, err)
	}

	tmpPath := localPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	if _, err := io.Copy(f, out.Body); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write %s: %w", localPath, err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, localPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

// DeleteObject deletes an object from R2 (succeeds if key does not exist).
func (c *R2Client) DeleteObject(ctx context.Context, key string) error {
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFoundErr(err) {
			return nil
		}
		return fmt.Errorf("delete %s: %w", key, err)
	}
	return nil
}

// PresignGetURL generates a time-limited S3 presigned download URL for key.
func (c *R2Client) PresignGetURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	if expiry <= 0 {
		expiry = 24 * time.Hour
	}
	presigner := s3.NewPresignClient(c.client)
	out, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("presign %s: %w", key, err)
	}
	return out.URL, nil
}

// PutJSON marshals payload as JSON and stores it at key.
func (c *R2Client) PutJSON(ctx context.Context, key string, payload any) error {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json for %s: %w", key, err)
	}

	_, err = c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return fmt.Errorf("put json %s: %w", key, err)
	}
	return nil
}

// GetJSON retrieves JSON from key. Returns (false, nil) if the key does not exist.
func (c *R2Client) GetJSON(ctx context.Context, key string, out any) (bool, error) {
	resp, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFoundErr(err) {
			return false, nil
		}
		return false, fmt.Errorf("get json %s: %w", key, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("read json %s: %w", key, err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return false, nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return false, fmt.Errorf("decode json %s: %w", key, err)
	}
	return true, nil
}

// ObjectExists checks whether key exists in the R2 bucket.
func (c *R2Client) ObjectExists(ctx context.Context, key string) (bool, error) {
	_, err := c.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFoundErr(err) {
			return false, nil
		}
		return false, fmt.Errorf("head %s: %w", key, err)
	}
	return true, nil
}

func isNotFoundErr(err error) bool {
	var noSuchKey *types.NoSuchKey
	if errors.As(err, &noSuchKey) {
		return true
	}
	var notFound *types.NotFound
	if errors.As(err, &notFound) {
		return true
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		if code == "NoSuchKey" || code == "NotFound" || code == "404" {
			return true
		}
	}
	var respErr interface{ HTTPStatusCode() int }
	if errors.As(err, &respErr) && respErr.HTTPStatusCode() == http.StatusNotFound {
		return true
	}
	return false
}

