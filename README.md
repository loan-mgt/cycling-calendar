<div align="center">
  <img width=200 alt="logo cycling-calendar" src="/static/favicon.svg">

  # Cycling calendar

  Never miss a cycling race again.

</div>

# About

The goal of the CPE Calendar is to offer students easy and effortless access to their school schedule by syncing it with their personal calendar. This works on all devices (phones, computers, laptops) and with any calendar provider, including Apple and Google.

The calendar automatically updates every hour, keeping you informed of any schedule changes. This project is open-source, and contributions or issue reports are welcome on GitHub.

# Setup

1. The first step is to set up your own `.env` file. Use `example.env` as a reference.
2. Then run the production version using Docker Compose.

# Usage

Warning the info is base on 'https://www.procyclingstats.com/races.php?timezone=fr&filter=Filter&p=uci&s=start-finish-schedule'

# Development

If you want to run the project without the Docker environment, follow these steps:

### Start the code
```bash
go mod download
go run main.go
```

# Affiliation

This project is entirely independent and is not affiliated with any organization.
