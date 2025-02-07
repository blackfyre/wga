{ pkgs, lib, config, inputs, ... }:

{

  env.WGA_ADMIN_EMAIL="some.random.email@local.host";    # admin email address
  env.WGA_ADMIN_PASSWORD="VerySecurePassword";           # admin password

  env.WGA_S3_ENDPOINT="http://localhost:9000";                         # minio server in docker, change for production!
  env.WGA_S3_BUCKET="wga";                                             # default minio bucket in docker, change for production!
  env.WGA_S3_REGION="eu-west-1";                                                # empty for minio, change for production!
  env.WGA_S3_ACCESS_KEY="WKQPUXGRDXWUCEIGJVBZ";                        # default minio access key in docker, change for production!
  env.WGA_S3_ACCESS_SECRET="wAeifxp7TpJy17u9fxRgJ6ONXCvZfi90qs3j9z1i"; # default minio secret key in docker, change for production!

  env.WGA_PROTOCOL="http";           # http or https
  env.WGA_HOSTNAME="localhost:8090"; # hostname (and port)

  env.WGA_SMTP_HOST="127.0.0.1";      # smtp server
  env.WGA_SMTP_PORT="8025";      # smtp port
  env.WGA_SMTP_USERNAME="";  # smtp username
  env.WGA_SMTP_PASSWORD="";  # smtp password
  env.WGA_SENDER_ADDRESS="do-not-reply@wga.hu"; # sender email address
  env.WGA_SENDER_NAME="WGA";    # sender name

  env.MAILPIT_URL="http://127.0.0.1:8025"; # mailpit url

  services.minio.accessKey = "minio";
  services.minio.secretKey = "minio123";
}
