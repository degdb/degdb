FROM golang:onbuild
EXPOSE 8080 7946
ENV persist_dir /opt/degdb_data
WORKDIR ${persist_dir}
