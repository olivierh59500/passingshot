export GOCACHE := $(CURDIR)/.cache/go-build

.PHONY: extract run build android android-run presentation test fmt

extract:
	go run ./cmd/extract -adf "$(ADF)"

run:
	go run ./cmd/passingshot

build:
	mkdir -p build/desktop
	go build -trimpath -o build/desktop/passingshot ./cmd/passingshot

android:
	./scripts/build-android.sh

android-run:
	./scripts/run-android.sh

presentation:
	go run ./cmd/presentation

test:
	go test ./...
	go vet ./...

fmt:
	gofmt -w cmd internal mobile
