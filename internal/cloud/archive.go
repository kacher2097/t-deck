package cloud

import (
	"archive/zip"
	"compress/flate"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func zipSource(sourcePath string, destZip string) error {
	return zipSourceFast(sourcePath, destZip, nil)
}

func zipSourceFast(sourcePath string, destZip string, progressCb func(done, total int64)) error {
	outFile, err := os.Create(destZip)
	if err != nil {
		return err
	}
	defer outFile.Close()

	zw := zip.NewWriter(outFile)
	zw.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(out, flate.BestSpeed)
	})
	defer zw.Close()

	info, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		err := addFileToZipFast(zw, sourcePath, filepath.Base(sourcePath))
		if err == nil && progressCb != nil {
			progressCb(info.Size(), info.Size())
		}
		return err
	}

	var totalBytes int64
	var files []struct {
		path string
		rel  string
		size int64
	}
	if err := filepath.WalkDir(sourcePath, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			lower := strings.ToLower(d.Name())
			if path != sourcePath && (lower == ".git" || lower == "node_modules" || lower == "__pycache__" || lower == ".venv") {
				return filepath.SkipDir
			}
			return nil
		}
		lower := strings.ToLower(d.Name())
		if strings.HasSuffix(lower, ".pdb") || strings.HasSuffix(lower, ".log") {
			return nil
		}
		rel, err := filepath.Rel(sourcePath, path)
		if err != nil {
			return err
		}
		fInfo, err := d.Info()
		if err != nil {
			return err
		}
		totalBytes += fInfo.Size()
		files = append(files, struct {
			path string
			rel  string
			size int64
		}{path: path, rel: filepath.ToSlash(rel), size: fInfo.Size()})
		return nil
	}); err != nil {
		return err
	}

	var doneBytes int64
	for _, f := range files {
		if err := addFileToZipFast(zw, f.path, f.rel); err != nil {
			return err
		}
		doneBytes += f.size
		if progressCb != nil {
			progressCb(doneBytes, totalBytes)
		}
	}
	return nil
}

func isAlreadyCompressedExt(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".exe", ".msi", ".zip", ".gz", ".7z", ".rar", ".asar", ".png", ".jpg", ".jpeg", ".webp", ".mp4", ".woff2":
		return true
	default:
		return false
	}
}

func addFileToZipFast(zw *zip.Writer, srcPath string, archiveName string) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = archiveName
	if isAlreadyCompressedExt(srcPath) {
		header.Method = zip.Store
	} else {
		header.Method = zip.Deflate
	}

	w, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, f)
	return err
}

func unzipSecure(zipPath string, destDir string) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer zr.Close()

	cleanDest := filepath.Clean(destDir) + string(os.PathSeparator)

	for _, f := range zr.File {
		targetPath := filepath.Join(destDir, filepath.FromSlash(f.Name))
		cleanTarget := filepath.Clean(targetPath)
		if !strings.HasPrefix(cleanTarget, cleanDest) && cleanTarget != filepath.Clean(destDir) {
			return fmt.Errorf("illegal zip entry path: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(cleanTarget, 0o755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(cleanTarget), 0o755); err != nil {
			return err
		}
		if err := extractZipEntry(f, cleanTarget); err != nil {
			return err
		}
	}
	return nil
}

func extractZipEntry(f *zip.File, destPath string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, rc)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

