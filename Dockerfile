FROM registry.redhat.io/ubi8/go-toolset

WORKDIR /go/src/app
COPY . .

USER 0

RUN go get -d ./... && \
    go build -o /bin/rhc_worker_catalog

#RUN cp /opt/app-root/src/go/bin/rhc_worker_catalog /usr/bin/

RUN yum remove -y kernel-headers npm nodejs nodejs-full-i18n && yum update -y && yum clean all

USER 1001
CMD ["rhc_worker_catalog"]
