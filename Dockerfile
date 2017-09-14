FROM ubuntu
ADD app .
ENTRYPOINT ["./app"]
