These variables must be defined in ``vars/main.yaml``::

  # Find these from your Cloudflare account dashboard when you create an API key
  cf_api_token:
  cf_account_id:
  cf_zone_id:
  domain_name: fabrikam.com

  install_path: /opt/dynamic-dns-service

  # How often the cron job runs
  interval: 5m

  # Architecture of the server this will run on (e.g., amd64 or arm64)
  target_arch: amd64
