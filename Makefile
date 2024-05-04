

NAME=gh-flox

local: fmt
	go build -o $(NAME) .

fmt: 
	go fmt .

ready: fmt
	GOOS=linux go build .
	scp $(NAME) @bot:
	scp $(NAME).desc @bot:
	ssh bot "sudo mv /home/ec2-user/$(NAME) /var/lib/goldiflox/shell/$(NAME) && mv /home/ec2-user/$(NAME).desc /var/lib/goldiflox/shell/$(NAME).desc"

