FROM scratch

COPY ca-bundle.crt /etc/ssl/certs/ca-certificates.crt
COPY votehub /
COPY */templates/*.html /templates/

ENV TEMPLATES=/templates/*.html

ENTRYPOINT ["/votehub"]
CMD [""]
