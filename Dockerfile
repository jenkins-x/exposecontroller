FROM scratch

ENTRYPOINT ["/exposecontroller"]

COPY ./bin/exposecontroller-docker /exposecontroller
