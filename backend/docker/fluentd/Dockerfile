FROM fluent/fluentd:v1.16-1

USER root
# Pin elasticsearch ruby gem to 7.17.x — plugin 5.4.3's compatible-with
# header parser breaks against the ES 8.x client default. See
# uken/fluent-plugin-elasticsearch#1027.
RUN gem install elasticsearch -v 7.17.11 --no-document \
 && gem install elasticsearch-api -v 7.17.11 --no-document \
 && gem install elasticsearch-transport -v 7.17.11 --no-document \
 && gem install fluent-plugin-elasticsearch -v 5.4.3 --no-document
USER fluent
