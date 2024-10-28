{ pkgs, lib, config, inputs, ... }:

{
  services.minio.accessKey = "minio";
  services.minio.secretKey = "minio123";
}
