# CockroachDB CDC to Google Pub/Sub Bridge

This application demonstrates an approach to connecting a [CockroachDB
Enterprise Change Data
Capture](https://www.cockroachlabs.com/docs/v20.2/stream-data-out-of-cockroachdb-using-changefeeds.html#configure-a-changefeed-enterprise)
(CDC) feed into [Google's
Pub/Sub](https://cloud.google.com/pubsub/docs/overview) service.

This uses the experimental HTTP backend to deliver JSON-formatted
payloads to a topic.

## Getting Started

* Create a GCP service account and download its JSON credentials file.
* Grant the service account `Pub/Sub Editor` to automatically create a
  topic, or `Pub/Sub Publisher` if you wish to manually create the topic.
* Move the JSON credentials file into a working directory `$HOME/cdc-pubsub/cdc-pubsub.json`
* Start the bridge server:
    * `docker run --rm -it -v $HOME/cdc-pubsub:/data:ro -p 13013:13013 bobvawter/cdc-pubsub:latest --projectID my-project-id --sharedKey xyzzy`
* Create an enterprise changefeed in CockroachDB:
    * `CREATE CHANGEFEED FOR TABLE foo INTO 'experimental-http://127.0.0.1:13013/v1/my-topic?sharedKey=xyzzy' WITH updated;`
    * Replace `my-topic` with your preferred topic name.
* Check the log for progress.

## Pub/Sub Attributes

Each Pub/Sub message will be labelled with the following attributes.

* `table`: The affected SQL table.
* `path`: The complete path used to post the message.

## Building

`docker build . -t cdc-pubsub`

## Other endpoints

If the bridge is to be placed behind a load-balancer (e.g. in a
Kubernetes environment), there is a `/healthz` endpoint which always
returns `OK`.

Runtime profiling information is available at `/debug/pprof`

## Security implications

The bridge server relies upon having a shared key that is provided by
the CDC feed via the `sharedKey` query parameter. Any client with this
shared key can effectively post arbitrary messages to the topic.

Seamless rotation of shared keys is possible by passing multiple
`--sharedKey` arguments to the bridge server.

Google Cloud IAM restrictions can be added to the role account to limit
the names of the Pub/Sub topics that it may access.