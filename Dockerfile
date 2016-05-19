FROM centos:7

ENV PATH $PATH:/usr/local/exposer/

ADD ./bin/exposer /usr/local/exposer/

CMD exposer
