{ pkgs, lib, config, inputs, ... }:

{

  # https://devenv.sh/packages/
  packages = [
    pkgs.git
    pkgs.templ
    pkgs.air
  ]  ++ lib.optionals (!config.container.isBuilding) [
    pkgs.flyctl
    pkgs.nil
  ];

  # https://devenv.sh/languages/
  # languages.rust.enable = true;

  languages.go.enable = true;
  languages.go.enableHardeningWorkaround = true;

  languages.javascript = {
    enable = true;
    bun = {
      enable = true;
      install.enable = true;
    };
  };

  services.mailhog.enable = true;

  services.minio.enable = true;
  services.minio.buckets = [
    "wga"
  ];

  enterShell = ''
    bun --version
    git --version
    go version
  '';

  scripts.generate-templates.exec = "templ generate";
  scripts.tidy-modules.exec = "go mod tidy";
  scripts.tidy.exec = ''
    devenv shell generate-templates
    devenv shell tidy-modules
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
