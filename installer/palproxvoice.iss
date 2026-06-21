; PalProxVoice — instalador (Inno Setup)
; Gera PalProxVoice-Setup.exe: acha o Palworld, instala UE4SS+mod no jogo,
; instala o companion+config, e configura auto-start (minimizado).

[Setup]
AppName=PalProxVoice
AppVersion=0.5
AppPublisher=Fernando Braga
DefaultDirName={code:DetectPalworld}
AppendDefaultDirName=no
DirExistsWarning=no
DisableProgramGroupPage=yes
PrivilegesRequired=admin
OutputBaseFilename=PalProxVoice-Setup
OutputDir=.
Compression=lzma2
SolidCompression=yes
WizardStyle=modern

[Languages]
Name: "br"; MessagesFile: "compiler:Languages\BrazilianPortuguese.isl"

[Files]
; companion + config -> LocalAppData do usuario
Source: "palproxvoice.exe"; DestDir: "{localappdata}\PalProxVoice"; Flags: ignoreversion
Source: "config.json"; DestDir: "{localappdata}\PalProxVoice"; Flags: onlyifdoesntexist
; UE4SS v3.0.1 + mod -> a pasta Binaries certa do jogo (Win64 ou WinGDK, em runtime)
Source: "ue4ss\*"; DestDir: "{code:GetBinDir}"; Flags: recursesubdirs createallsubdirs ignoreversion

[Icons]
; auto-start: sobe minimizado com o Windows
Name: "{userstartup}\PalProxVoice"; Filename: "{localappdata}\PalProxVoice\palproxvoice.exe"; Parameters: "-min"; WorkingDir: "{localappdata}\PalProxVoice"

[Run]
Filename: "{localappdata}\PalProxVoice\palproxvoice.exe"; Description: "Abrir o PalProxVoice agora"; Flags: nowait postinstall skipifsilent

[Code]
function SteamPalworld(): String;
var steam: String;
begin
  Result := '';
  if RegQueryStringValue(HKCU, 'Software\Valve\Steam', 'SteamPath', steam) then
  begin
    StringChangeEx(steam, '/', '\', True);
    if DirExists(steam + '\steamapps\common\Palworld\Pal\Binaries') then
      Result := steam + '\steamapps\common\Palworld';
  end;
end;

function DetectPalworld(Param: String): String;
begin
  Result := SteamPalworld();
end;

function GetBinDir(Param: String): String;
begin
  if DirExists(ExpandConstant('{app}') + '\Pal\Binaries\WinGDK') then
    Result := ExpandConstant('{app}') + '\Pal\Binaries\WinGDK'
  else
    Result := ExpandConstant('{app}') + '\Pal\Binaries\Win64';
end;

// valida que a pasta escolhida e o Palworld
function NextButtonClick(CurPageID: Integer): Boolean;
begin
  Result := True;
  if CurPageID = wpSelectDir then
    if not DirExists(ExpandConstant('{app}') + '\Pal\Binaries') then
    begin
      MsgBox('Essa pasta nao parece ser o Palworld (falta Pal\Binaries). Escolha a pasta raiz do Palworld.', mbError, MB_OK);
      Result := False;
    end;
end;
