TAG?=latest
IMAGE_NAME=carlosdescalzi/jobgraphs:$(TAG)

all: test image

test:
	go test ./... -coverprofile cover.out
	if [ "$(DISPLAY)" != "" ]; then \
		go tool cover -html=cover.out; \
	fi

build:
	CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w " -a -installsuffix 'static' -o ./jobgraphs ./cmd/*

image: 
	docker build -t $(IMAGE_NAME) $(PWD)

run: 
	go run cmd/* -kubeconfig=$(KUBECONFIG) 
