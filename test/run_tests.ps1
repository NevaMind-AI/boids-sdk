#!/usr/bin/env pwsh
$ErrorActionPreference = "Stop"

$TestDir = $PSScriptRoot

function Resolve-TestCommand {
  param(
    [string]$EnvName,
    [string[]]$Names,
    [switch]$Optional
  )

  $override = [Environment]::GetEnvironmentVariable($EnvName)
  if ($override) {
    return $override
  }

  foreach ($name in $Names) {
    $command = Get-Command $name -ErrorAction SilentlyContinue
    if ($command) {
      return $command.Source
    }
  }

  if ($Optional) {
    return $null
  }

  throw "Could not find $($Names -join ', '). Set $EnvName to the executable path."
}

$Python = Resolve-TestCommand -EnvName "PYTHON" -Names @("python", "py")
$Node = Resolve-TestCommand -EnvName "NODE" -Names @("node")
$Go = Resolve-TestCommand -EnvName "GO" -Names @("go") -Optional

& $Python (Join-Path $TestDir "python_chat_complete.py")
& $Python (Join-Path $TestDir "python_response.py")
& $Node (Join-Path $TestDir "js_chat_complete.mjs")
& $Node (Join-Path $TestDir "js_response.mjs")

if ($Go) {
  Push-Location $TestDir
  try {
    & $Go run .\go_chat_complete.go
    & $Go run .\go_response.go
  }
  finally {
    Pop-Location
  }
}
else {
  Write-Warning "Skipping Go tests because Go was not found. Set GO to run them."
}
