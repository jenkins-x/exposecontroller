FROM scratch

ENTRYPOINT ["/exposecontroller", "--daemon"]

COPY ./out/exposecontroller-linux-amd64 /exposecontroller
