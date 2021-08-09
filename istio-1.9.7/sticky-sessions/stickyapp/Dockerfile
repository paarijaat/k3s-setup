FROM golang as build-stage
RUN mkdir src/stickyapp
COPY go.mod src/stickyapp/
COPY main.go src/stickyapp/
RUN cd src/stickyapp && CGO_ENABLED=0 GOOS=linux go build -v -a -o stickyapp

FROM paarijaat/ubuntu
COPY --from=build-stage /go/src/stickyapp/stickyapp /opt/
EXPOSE 8080
CMD ["/opt/stickyapp"]