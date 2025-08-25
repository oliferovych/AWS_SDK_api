SRCS := lambda.go
BINARY := main
ARCHIVE := main.zip

all: build compress

build:
	set GOOS=linux&& set GOARCH=amd64&& go build -o $(BINARY) $(SRCS)

compress:
	tar -a -c -f $(ARCHIVE) $(BINARY)

clean:
	del /f $(BINARY)

fclean:
	del /f $(BINARY) $(ARCHIVE)
