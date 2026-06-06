#!/usr/bin/env pwsh
$ErrorActionPreference = "Stop"

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")

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

& $Python (Join-Path $RepoRoot "test\python_chat_complete.py")
& $Node (Join-Path $RepoRoot "test\js_chat_complete.mjs")

if ($Go) {
  Push-Location (Join-Path $RepoRoot "test")
  try {
    & $Go run .\go_chat_complete.go
  }
  finally {
    Pop-Location
  }
}
else {
  Write-Warning "Skipping Go chat/complete test because Go was not found. Set GO to run it."
}
