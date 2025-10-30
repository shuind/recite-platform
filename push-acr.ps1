# push-acr.ps1 (ASCII / English)
# Usage: from project root run: .\push-acr.ps1

# --------- Edit these values ----------
$registry  = "crpi-b25bglrjfsh86jh6.cn-beijing.personal.cr.aliyuncs.com"
$namespace = "shuind"
$acrUser   = "shuind"
$backendDir  = ".\backend"
$frontendDir = ".\frontend"
$tag = "demo"
# --------------------------------------

Write-Host "ACR Registry: $registry"
Write-Host "Namespace: $namespace"

# prompt for password securely
$securePwd = Read-Host "ok" -AsSecureString
$ptr = [System.Runtime.InteropServices.Marshal]::SecureStringToBSTR($securePwd)
$plainPwd = [System.Runtime.InteropServices.Marshal]::PtrToStringAuto($ptr)
[System.Runtime.InteropServices.Marshal]::ZeroFreeBSTR($ptr)

try {
    Write-Host "`n-> Logging in to ACR..." -ForegroundColor Yellow
    $plainPwd | docker login $registry -u $acrUser --password-stdin
    if ($LASTEXITCODE -ne 0) { throw "docker login failed" }

    Write-Host "`n-> Building backend image..." -ForegroundColor Yellow
    docker build -t recite-backend:local $backendDir
    if ($LASTEXITCODE -ne 0) { throw "backend build failed" }

    $backendRemote = "$registry/$namespace/recite-backend:$tag"
    docker tag recite-backend:local $backendRemote
    docker push $backendRemote
    if ($LASTEXITCODE -ne 0) { throw "backend push failed" }

    Write-Host "`n-> Building frontend image..." -ForegroundColor Yellow
    docker build -t recite-frontend:local $frontendDir
    if ($LASTEXITCODE -ne 0) { throw "frontend build failed" }

    $frontendRemote = "$registry/$namespace/recite-frontend:$tag"
    docker tag recite-frontend:local $frontendRemote
    docker push $frontendRemote
    if ($LASTEXITCODE -ne 0) { throw "frontend push failed" }

    Write-Host "`n-> Pulling images to verify..." -ForegroundColor Yellow
    docker pull $backendRemote
    docker pull $frontendRemote

    Write-Host "`nAll done. Images pushed:" -ForegroundColor Green
    Write-Host "  $backendRemote"
    Write-Host "  $frontendRemote"
}
catch {
    Write-Host "`nError: $_" -ForegroundColor Red
}
finally {
    $plainPwd = $null
    $securePwd = $null
}
