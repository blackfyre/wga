{ pkgs, lib, config, inputs, ... }:

{

  # https://devenv.sh/packages/
  packages = [
    pkgs.git
    pkgs.nil
    pkgs.templ
    pkgs.flyctl
  ];

  # https://devenv.sh/languages/
  # languages.rust.enable = true;

  languages.go.enable = true;
  languages.go.enableHardeningWorkaround = true;

  languages.javascript.enable = true;
  languages.javascript.bun.enable = true;

  services.mailhog.enable = true;

  services.minio.enable = true;
  services.minio.buckets = [
    "wga"
  ];
  services.minio.accessKey = "minio";
  services.minio.secretKey = "minio123";

  enterShell = ''
    bun --version
    git --version
    go version
  '';

  scripts.tidy.exec = ''
    templ generate && go mod tidy
  '';

  pre-commit.hooks = {
    govet = {
      enable = true;
      pass_filenames = false;
    };
    gotest.enable = true;
    golangci-lint = {
      enable = true;
      pass_filenames = false;
    };
  };

  # See full reference at https://devenv.sh/reference/options/
}
