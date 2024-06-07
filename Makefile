

NAME=gh-flox
LDFLAGS=-ldflags "-X main.GitSHA=$(shell git rev-parse HEAD) -X main.GitDirty=$(shell git diff --quiet || echo 'dirty')"

local: fmt
	go build $(LDFLAGS) -o $(NAME) .

fmt: 
	go fmt .

ready: fmt
	GOOS=linux go build $(LDFLAGS) -o $(NAME) .
	scp $(NAME) @bot:
	scp $(NAME).desc @bot:
	ssh bot "sudo mv /home/ec2-user/$(NAME) /var/lib/goldiflox/shell/$(NAME) && mv /home/ec2-user/$(NAME).desc /var/lib/goldiflox/shell/$(NAME).desc"

lambda: fmt
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bootstrap .
	zip bootstrap.zip  bootstrap
	# Update existing lambda function
	aws lambda update-function-code --function-name $(NAME)  --zip-file fileb://bootstrap.zip

clean:
	rm -f $(NAME) bootstrap bootstrap.zip
