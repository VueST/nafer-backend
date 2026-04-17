Write-Host "=======================================" -ForegroundColor Cyan
Write-Host "  Starting Nafer Native Backend Cluster" -ForegroundColor Cyan
Write-Host "=======================================" -ForegroundColor Cyan

# 1. Inject Environment Variables into this Terminal Session
Write-Host ">> Injecting Environment Variables (Localhost Mode)..." -ForegroundColor Yellow
$env:DATABASE_URL="postgres://nafer_admin:nafer_password@localhost:5432/nafer_db?sslmode=disable"
$env:REDIS_URL="localhost:6379"
$env:MINIO_ENDPOINT="localhost:9000"
$env:MINIO_ROOT_USER="nafer_storage_admin"
$env:MINIO_ROOT_PASSWORD="nafer_storage_password"
$env:MEDIA_BUCKET="nafer-media"
$env:STREAMING_BUCKET="nafer-streaming"
$env:MEILI_URL="http://localhost:7700"
$env:MEILI_KEY="nafer_master_key"
$env:JWT_SECRET="nafer_super_secret_jwt_key_that_is_long_enough_32_chars"
$env:JWT_EXPIRES_IN="24h"

# 2. Kill any previously running Go services
Stop-Process -Name "go" -ErrorAction SilentlyContinue
Stop-Process -Name "main" -ErrorAction SilentlyContinue

# 3. Start all Microservices concurrently
Write-Host ">> Spinning up Go Microservices natively..." -ForegroundColor Yellow

$services = [ordered]@{
    "identity-service"     = @{ Port = "8001"; Path = "cmd/server/main.go" }
    "media-service"        = @{ Port = "8002"; Path = "cmd/server/main.go" }
    "comment-service"      = @{ Port = "8003"; Path = "cmd/server/main.go" }
    "notification-service" = @{ Port = "8004"; Path = "cmd/server/main.go" }
    "search-service"       = @{ Port = "8005"; Path = "cmd/server/main.go" }
    "streaming-service"    = @{ Port = "8006"; Path = "cmd/api/main.go" }
}

foreach ($service in $services.GetEnumerator()) {
    $svcName = $service.Key
    $svcPort = $service.Value.Port
    $svcPath = $service.Value.Path
    
    Write-Host "  -> Starting $svcName on PORT $svcPort (in background)..." -ForegroundColor Green
    
    # We MUST set the PORT explicitly so the native Fiber servers don't crash colliding on 8080!
    $env:PORT = $svcPort
    
    Start-Process -FilePath "go" -ArgumentList "run", $svcPath -WorkingDirectory ".\$svcName" -NoNewWindow
}

Write-Host "=======================================" -ForegroundColor Cyan
Write-Host "  All Services are running perfectly!  " -ForegroundColor Green
Write-Host "  To stop them all later, run this:    " -ForegroundColor Yellow
Write-Host "  taskkill /F /IM go.exe /IM main.exe  " -ForegroundColor White
Write-Host "=======================================" -ForegroundColor Cyan
