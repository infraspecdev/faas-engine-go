#define VERSION "0.0.0"  ; 

[Setup]
AppName=Nimbus CLI
AppVersion={#VERSION}
DefaultDirName={localappdata}\Nimbus
DefaultGroupName=Nimbus
OutputDir=Output
OutputBaseFilename=nimbus-installer
Compression=lzma
SolidCompression=yes
PrivilegesRequired=lowest

[Files]
Source: "nimbus.exe"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\Nimbus CLI"; Filename: "{app}\nimbus.exe"

[Code]
procedure CurStepChanged(CurStep: TSetupStep);
var
  Path: string;
  NewPath: string;
  ResultCode: Integer;
begin
  if CurStep = ssPostInstall then
  begin
    if RegQueryStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', Path) then
    begin
      if Pos(ExpandConstant('{app}'), Path) = 0 then
      begin
        if Path <> '' then
          NewPath := Path + ';' + ExpandConstant('{app}')
        else
          NewPath := ExpandConstant('{app}');

        RegWriteStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', NewPath);

        { Refresh environment variables immediately }
        SendMessageTimeout(HWND_BROADCAST, WM_SETTINGCHANGE, 0,
          Integer(PChar('Environment')), SMTO_ABORTIFHUNG, 5000, ResultCode);
      end;
    end;

    MsgBox('Nimbus installed! Restart terminal and run: nimbus', mbInformation, MB_OK);
  end;
end;