$ErrorActionPreference = 'Stop'

$toolsDir = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"

$packageArgs = @{
  packageName    = $env:ChocolateyPackageName
  fileFullPath   = Join-Path $toolsDir 'actup.exe'
  url64bit       = 'https://github.com/lynicis/actup/releases/download/v{{VERSION}}/gh-actup-windows-amd64.exe'
  checksum64     = '{{CHECKSUM64}}'
  checksumType64 = 'sha256'
}

Get-ChocolateyWebFile @packageArgs
