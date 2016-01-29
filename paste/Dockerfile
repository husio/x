FROM scratch

COPY ca-bundle.crt /etc/ssl/certs/ca-certificates.crt
COPY /public /
COPY paste /

ENTRYPOINT ["/paste"]
CMD [""]
