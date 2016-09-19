FROM scratch

ENTRYPOINT ["/exposecontroller"]

COPY ./out/exposecontroller-linux-amd64 /exposecontroller
