<div align="center">
  <img width=200 alt="logo cycling-calendar" src="/static/favicon.svg">

  # Cycling calendar

  Never miss a cycling race again.

</div>

# About

Get the estimated end time of each cycling race directly in your calendar!

# Setup

1. The first step is to set up your own `.env` file. Use `example.env` as a reference.
2. Then run the production version using Docker Compose.

# Usage

The info is based on 'https://cyclingtiz.live/'. The data is cached for 24 hours to reduce load on the source.

Supported categories include:
- **ME**: Men Elite
- **WE**: Women Elite
- **Track**: Track Cycling
- **MTB**: Mountain Biking
- **NC**: National Championships
- **WC**: World Championships
- **JR**: Junior

# Development

If you want to run the project without the Docker environment, follow these steps:

### Start the code
```bash
go mod download
go run .
```

# Affiliation

This project is entirely independent and is not affiliated with any organization.
