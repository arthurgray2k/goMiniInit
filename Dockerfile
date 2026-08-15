FROM alpine:latest

COPY gominiinit /goMiniInit
COPY examples /examples

ENTRYPOINT ["/goMiniInit"]
CMD ["sleep", "1000"]
