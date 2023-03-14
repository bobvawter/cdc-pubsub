# CockroachDB CDC to Google Pub/Sub Bridge

CockroachDB as of v22.1 [natively supports](https://www.cockroachlabs.com/docs/v22.1/create-changefeed#create-a-changefeed-connected-to-a-google-cloud-pub-sub) sending a changefeed to Google Pub/Sub. This repository is now archived, but will be retained for demonstration purposes.

This application demonstrates an approach to connecting a [CockroachDB
Enterprise Change Data
Capture](https://www.cockroachlabs.com/docs/v20.2/stream-data-out-of-cockroachdb-using-changefeeds.html#configure-a-changefeed-enterprise)
(CDC) feed into [Google's
Pub/Sub](https://cloud.google.com/pubsub/docs/overview) service,
until such time as CockroachDB
[natively supports](https://github.com/cockroachdb/cockroach/issues/36982)
Google Pub/Sub in a future release.

This uses the experimental HTTP(S) backend to deliver JSON-formatted
payloads to a topic.

## Getting Started

* Create a GCP service account and download its JSON credentials file.
* Grant the service account `Pub/Sub Editor` to automatically create a
  topic, or `Pub/Sub Publisher` if you wish to manually create the topic.
* Move the JSON credentials file into a working directory `$HOME/cdc-pubsub/cdc-pubsub.json`
* Start the bridge server:
    * `docker run --rm -it -v $HOME/cdc-pubsub:/data:ro -p 13013:13013 bobvawter/cdc-pubsub:latest --projectID my-project-id --sharedKey xyzzy`
* Create an enterprise changefeed in CockroachDB:
    * `SET CLUSTER STETING kv.rangefeed.enabled = true;` if you haven't previously enabled rangefeeds for your cluster.
    * `CREATE CHANGEFEED FOR TABLE foo INTO 'experimental-http://127.0.0.1:13013/v1/my-topic?sharedKey=xyzzy' WITH updated;`
    * Replace `my-topic` with your preferred topic name.
* Check the log for progress.

## Flags

```
      --bindAddr string        the address to bind to (default ":13013")
      --credentials string     a JSON-formatted Google Cloud credentials file (default
                               "cdc-pubsub.json")
      --dumpOnly               if true, log payloads instead of sending to pub/sub
      --gracePeriod duration   shutdown grace period (default 30s)
  -h, --help                   display this message
      --projectID string       the Google Cloud project ID
      --sharedKey strings      require clients to provide one of these secret values
      --topicPrefix string     a prefix to add to topic names
```

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

The bridge server provides the option of shared key which is provided by
the CDC feed via the `sharedKey` query parameter. This key prevents
users from inadvertently "crossing the streams" as opposed to being a
proper security mechanism:
* Any HTTP client with this shared key can effectively post arbitrary
  messages to any Pub/Sub topic that the bridge's service account has
  access to.
* Any SQL user that can execute the `SHOW JOBS` command can view the shared key.
* Any user that can view the Jobs page in the Admin UI can view the shared key.
* The shared key will likely appear unobfuscated in CockroachDB logs.

Seamless rotation of shared keys is possible by passing multiple
`--sharedKey` arguments to the bridge server.

Google Cloud IAM restrictions can be added to the role account to limit
the names of the Pub/Sub topics that it may access.

## Deployment strategy

Given the lightweight nature of the bridge server and the above security
limitations, users should deploy this server as a "sidecar" alongside
each of their CockroachDB nodes, bound only to a loopback IP address via
the `--bindAddr` flag.

If the bridge is to be deployed as a traditional network service, it
should be placed behind a TLS loadbalancer with appropriate firewall
rules.
