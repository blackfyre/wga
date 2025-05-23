{ pkgs, lib, config, inputs, ... }:

{

  # https://devenv.sh/packages/
  packages = [
    pkgs.git
    pkgs.templ
    pkgs.air
    pkgs.nixd
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
    generate-templates
    tidy-modules
  '';
  scripts."app:build".exec = "
    mkdir -p dist;
    rm -rf dist/app;
    bun install;
    bun run build;
    tidy;
    go build -v -o dist/app .;";

  scripts."app:run".exec = "./dist/app serve";

  scripts."app:reboot".exec = ''
    app:build;
    rm -rf wga_data;
    app:run;
  '';
  
  scripts.init-devenv.exec = "cp devenv.local.stub.nix devenv.local.nix";
  git-hooks.hooks = {
    govet = {
      enable = true;
      pass_filenames = false;
    };
    #gotest.enable = true;
    golangci-lint = {
      enable = true;
      pass_filenames = false;
    };
  };

  processes = {
    watch-js.exec = "bun run build:watch:js";
    templ.exec = "templ generate --watch";
    # air.exec = "air serve --dev";
    watch-css.exec = "bun run build:watch:css";
  };

  # See full reference at https://devenv.sh/reference/options/
}
