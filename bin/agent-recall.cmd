@echo off
setlocal
set "ROOT=%~dp0.."

where go >nul 2>nul
if errorlevel 1 (
  echo agent-recall source marketplace install requires Go on PATH. Use a packaged release artifact for a bundled binary. 1>&2
  exit /b 127
)

cd /d "%ROOT%" || exit /b 1
go run ./cmd/agent-recall %*
