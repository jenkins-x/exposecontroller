FROM centos:7

ENV PATH $PATH:/usr/local/exposecontroller/

ADD ./bin/exposecontroller /usr/local/exposecontroller/

CMD exposecontroller
