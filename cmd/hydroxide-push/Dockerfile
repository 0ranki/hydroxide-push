FROM scratch

COPY hydroxide-push /
ENV HOME=/data
ENV HYDROXIDE_BRIDGE_PASS="supersecretpassword"
WORKDIR /data
ENTRYPOINT ["/hydroxide-push"]
CMD ["notify"]