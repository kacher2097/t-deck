Unicode true

!define INFO_PROJECTNAME    "TDeck"
!define INFO_COMPANYNAME    "T-Hub"
!define INFO_PRODUCTNAME    "T-Deck"
!define PRODUCT_EXECUTABLE  "TDeck.exe"
!define REQUEST_EXECUTION_LEVEL "user"
!define WAILS_INSTALL_SCOPE "user"

!include "wails_tools.nsh"

VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion    "${INFO_PRODUCTVERSION}.0"

VIAddVersionKey "CompanyName"     "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "Bộ cài đặt ${INFO_PRODUCTNAME}"
VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion"     "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright"  "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName"     "${INFO_PRODUCTNAME}"

ManifestDPIAware true

!include "MUI2.nsh"

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
!define MUI_FINISHPAGE_NOAUTOCLOSE
!define MUI_ABORTWARNING
!define MUI_COMPONENTSPAGE_SMALLDESC

!define MUI_WELCOMEPAGE_TITLE "Chào mừng đến với trình cài đặt ${INFO_PRODUCTNAME}"
!define MUI_WELCOMEPAGE_TEXT "Trình hướng dẫn sẽ cài đặt ${INFO_PRODUCTNAME} vào thư mục bạn chọn.$\r$\n$\r$\nToàn bộ dữ liệu ứng dụng (tdeck.db, settings.json, .env, thumbnails, apps) sẽ được lưu mặc định ngay trong thư mục cài đặt để dễ sao lưu và di chuyển.$\r$\n$\r$\nNhấn Tiếp theo để tiếp tục."

!define MUI_DIRECTORYPAGE_TEXT_TOP "Chọn thư mục cài đặt ${INFO_PRODUCTNAME}.$\r$\nDữ liệu cá nhân (tdeck.db, thumbnails/, settings.json, apps/) sẽ được lưu trực tiếp tại thư mục này thay vì ổ C."

!define MUI_FINISHPAGE_RUN "$INSTDIR\${PRODUCT_EXECUTABLE}"
!define MUI_FINISHPAGE_RUN_TEXT "Khởi chạy ${INFO_PRODUCTNAME} ngay bây giờ"
!define MUI_FINISHPAGE_SHOWREADME ""
!define MUI_FINISHPAGE_SHOWREADME_NOTCHECKED
!define MUI_FINISHPAGE_SHOWREADME_TEXT "Mở thư mục cài đặt ${INFO_PRODUCTNAME}"
!define MUI_FINISHPAGE_SHOWREADME_FUNCTION OpenInstallDir

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_COMPONENTS
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "Vietnamese"

Name "${INFO_PRODUCTNAME}"
OutFile "..\..\bin\${INFO_PROJECTNAME}-${ARCH}-installer.exe"
InstallDir "$LOCALAPPDATA\Programs\${INFO_PRODUCTNAME}"
InstallDirRegKey HKCU "${UNINST_KEY}" "InstallLocation"
ShowInstDetails show

Function .onInit
    !insertmacro wails.checkArchitecture
FunctionEnd

Function OpenInstallDir
    ExecShell "open" "$INSTDIR"
FunctionEnd

Section "Chương trình cốt lõi ${INFO_PRODUCTNAME} (Bắt buộc)" SecCore
    SectionIn RO
    !insertmacro wails.setShellContext

    ; Dừng TDeck.exe đang chạy trước khi ghi đè file; giữ nguyên dữ liệu hiện có.
    nsExec::ExecToLog 'taskkill /IM ${PRODUCT_EXECUTABLE} /F'

    !insertmacro wails.webview2runtime

    SetOutPath "$INSTDIR"
    !insertmacro wails.files
    !ifdef ARG_TDECK_SYNC_BINARY
        File "/oname=tdeck-sync.exe" "${ARG_TDECK_SYNC_BINARY}"
    !endif

    CreateDirectory "$INSTDIR\thumbnails"
    CreateDirectory "$INSTDIR\apps"

    SetRegView 64
    WriteRegStr HKCU "${UNINST_KEY}" "InstallLocation" "$INSTDIR"

    !insertmacro wails.associateFiles
    !insertmacro wails.associateCustomProtocols
    !insertmacro wails.writeUninstaller
SectionEnd

Section "Tạo biểu tượng ngoài Màn hình chính (Desktop)" SecDesktop
    !insertmacro wails.setShellContext
    CreateShortCut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}" "" "$INSTDIR\${PRODUCT_EXECUTABLE}" 0
SectionEnd

Section "Tạo lối tắt trong Start Menu" SecStartMenu
    !insertmacro wails.setShellContext
    CreateDirectory "$SMPROGRAMS\${INFO_PRODUCTNAME}"
    CreateShortCut "$SMPROGRAMS\${INFO_PRODUCTNAME}\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}" "" "$INSTDIR\${PRODUCT_EXECUTABLE}" 0
    CreateShortCut "$SMPROGRAMS\${INFO_PRODUCTNAME}\Gỡ cài đặt ${INFO_PRODUCTNAME}.lnk" "$INSTDIR\uninstall.exe" "" "$INSTDIR\uninstall.exe" 0
SectionEnd

Section /o "Khởi động ${INFO_PRODUCTNAME} cùng Windows" SecAutoStart
    SetRegView 64
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "${INFO_PROJECTNAME}" '"$INSTDIR\${PRODUCT_EXECUTABLE}"'
SectionEnd

Section "Thêm vào biến môi trường PATH người dùng" SecPath
    nsExec::ExecToLog "powershell.exe -NoProfile -ExecutionPolicy Bypass -Command $\"& { param($$t); $$d = [Environment]::GetEnvironmentVariable('Path', 'User'); $$p = @(); if ($$d) { $$p = $$d.Split(';') | Where-Object { $$_ -and ($$_.TrimEnd('\') -ine $$t.TrimEnd('\')) } }; [Environment]::SetEnvironmentVariable('Path', (($$p + $$t) -join ';'), 'User') }$\" $\"$INSTDIR$\""
SectionEnd

!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
    !insertmacro MUI_DESCRIPTION_TEXT ${SecCore} "Cài đặt ứng dụng ${INFO_PRODUCTNAME}, kiểm tra WebView2 Runtime và khởi tạo thư mục lưu trữ ngay tại thư mục cài đặt."
    !insertmacro MUI_DESCRIPTION_TEXT ${SecDesktop} "Tạo biểu tượng lối tắt ${INFO_PRODUCTNAME} trên màn hình Desktop."
    !insertmacro MUI_DESCRIPTION_TEXT ${SecStartMenu} "Tạo thư mục lối tắt trong Windows Start Menu kèm mục Gỡ cài đặt."
    !insertmacro MUI_DESCRIPTION_TEXT ${SecAutoStart} "Tự động mở ${INFO_PRODUCTNAME} mỗi khi bạn đăng nhập vào Windows."
    !insertmacro MUI_DESCRIPTION_TEXT ${SecPath} "Cho phép mở nhanh ${INFO_PRODUCTNAME} hoặc tdeck-sync từ Terminal / Command Prompt."
!insertmacro MUI_FUNCTION_DESCRIPTION_END

Section "uninstall"
    !insertmacro wails.setShellContext
    nsExec::ExecToLog 'taskkill /IM ${PRODUCT_EXECUTABLE} /F'

    Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"
    Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"
    RMDir /r "$SMPROGRAMS\${INFO_PRODUCTNAME}"

    SetRegView 64
    DeleteRegValue HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "${INFO_PROJECTNAME}"
    nsExec::ExecToLog "powershell.exe -NoProfile -ExecutionPolicy Bypass -Command $\"& { param($$t); $$d = [Environment]::GetEnvironmentVariable('Path', 'User'); if ($$d) { $$p = $$d.Split(';') | Where-Object { $$_ -and ($$_.TrimEnd('\') -ine $$t.TrimEnd('\')) } }; [Environment]::SetEnvironmentVariable('Path', ($$p -join ';'), 'User') }$\" $\"$INSTDIR$\""

    Delete "$INSTDIR\${PRODUCT_EXECUTABLE}"
    Delete "$INSTDIR\tdeck-sync.exe"

    !insertmacro wails.unassociateFiles
    !insertmacro wails.unassociateCustomProtocols
    !insertmacro wails.deleteUninstaller

    MessageBox MB_YESNO|MB_ICONQUESTION|MB_DEFBUTTON2 "Bạn có muốn xóa luôn toàn bộ dữ liệu cá nhân (tdeck.db, settings.json, .env, thumbnails, apps) trong thư mục cài đặt không?$\r$\n$\r$\nChọn No (Khuyên dùng) để giữ lại dữ liệu." IDNO keep_user_data
        RMDir /r "$INSTDIR"
        Goto done_uninstall
    keep_user_data:
        RMDir "$INSTDIR"
    done_uninstall:
SectionEnd
