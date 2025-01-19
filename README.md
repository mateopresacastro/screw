# Screw

A little project with `Go`, `Next.js`, `0Auth2.0`, `DB Sessions`, `Nginx`, `SQLite`, `WebSocket`, `FFmpeg`, `Docker`, and `Docker Compose`.

The audio streams to a `Go` `API` via `WebSocket` connections, where `FFmpeg` processes each stream in real-time (slowed + reverb). The processed audio is returned through the same connections and buffered client-side, where it's rendered as a waveform.

Implemented Google's `0Auth2.0` flow with `PKCE` from sractch.

## Overview

![Overview diagram](images/overview.png)

## Prerequisites

- Docker and Docker Compose
- (Optional) Google OAuth2.0 credentials

## Setup

1. Clone the repository
2. Copy the example env file:

   ```bash
       cp .env.example .env
   ```

3. (Optional) Configure your `OAuth2.0` credentials in `.env`
4. Start the application:

   ```bash
   make dev
   ```

## Grafana

To see Grafana:

1. Go to `localhost:8080/grafana`.
2. Log in with `admin` `admin`.
3. Create `password`.
4. Click on the burger menu on the left. Click `Connections` > `Data Sources` > + `Add new data sources`.
5. Click `Prometheus` from the list.
6. Set `http://prometheus:9090` in the `Connection` input.
7. Click `Save & test`.
8. Click on the burger menu again > `Dashboards` > `New` > `Import`.
9. Copy the contents of the file at the root of the repo: `go-process-grafana-dashboard.json`. Click `Load`. Paste.
10. Select `prometheus` as the data source.

You should see this:

![Grafana dashboard](images/grafana.png)

## Dashboard credits

You can check the original dashboard [here](https://grafana.com/grafana/dashboards/6671-go-processes/).
