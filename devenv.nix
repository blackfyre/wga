{
  pkgs,
  lib,
  config,
  ...
}:

let
  minioAccessKey = "minio_access_key";
  minioSecretKey = "minio_secret_key";
  minioRegion = "us-east-1";
  minioBucket = "wga";
  minioEndpoint = "http://localhost:9000";
in
{

  # https://devenv.sh/packages/
  packages = [
    pkgs.git
    pkgs.templ
    pkgs.air
    pkgs.nixd
  ]
  ++ lib.optionals (!config.container.isBuilding) [
    pkgs.flyctl
    pkgs.nil
  ];

  env.WGA_S3_BUCKET = "${minioBucket}";
  env.WGA_S3_REGION = "${minioRegion}";
  env.WGA_S3_ACCESS_KEY = "${minioAccessKey}";
  env.WGA_S3_ACCESS_SECRET = "${minioSecretKey}";
  env.WGA_S3_ENDPOINT = "${minioEndpoint}";

  # https://devenv.sh/languages/
  languages = {
    # rust.enable = true;
    go = {
      enable = true;
      enableHardeningWorkaround = true;
    };

    javascript = {
      enable = true;
      bun = {
        enable = true;
        install.enable = true;
      };
    };
  };

  services = {
    mailhog = {
      enable = true;
    };

    minio = {
      enable = true;
      accessKey = "${minioAccessKey}";
      secretKey = "${minioSecretKey}";
      buckets = [ "${minioBucket}" ];
      afterStart = ''
        echo "MinIO started"
        mc anonymous set download local/${minioBucket}
      '';
    };

  };

  enterShell = ''
    bun --version
    git --version
    go version
  '';

  scripts = {
    "app:generate-templates" = {
      description = "Generate Go templates from templ sources.";
      exec = "templ generate";
    };
    "app:tidy-modules" = {
      description = "Run go mod tidy to sync module dependencies.";
      exec = "go mod tidy";
    };
    "app:tidy" = {
      description = "Regenerate templates and tidy Go modules.";
      exec = ''
        app:generate-templates
        app:tidy-modules
      '';
    };
    "app:build" = {
      description = "Produce the dist binary with fresh assets and dependencies.";
      exec = ''
        mkdir -p dist
        rm -rf dist/*
        bun install
        bun run build
        app:tidy
        go build -v -o dist/wga ./cmd/wga
      '';
    };
    "app:run" = {
      description = "Run the built server binary in development mode.";
      exec = ''
        pushd dist
        ./wga serve --dev
        popd
      '';
    };
    "app:reboot" = {
      description = "Rebuild, reset local data, and restart the development server.";
      exec = ''
        app:build
        rm -rf wga_data
        app:run
      '';
    };
    "app:init-devenv" = {
      description = "Bootstrap a local devenv configuration from the stub.";
      exec = "cp devenv.local.stub.nix devenv.local.nix";
    };
    "code:run" = {
      exec = "go run ./cmd/wga --dev";
      description = "Run the application in development mode without building.";
    };
  };
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
    watch_js.exec = "bun run build:watch:js";
    templ = {
      exec = "templ generate --watch";
      process-compose = {
        ready_log_line = "(✓) Watching files";
      };
    };
    # air = {
    #   exec = "air serve --dev";
    #   process-compose = {
    #     depends_on = {
    #       watch_js = {
    #         condition = "process_started";
    #       };
    #       templ = {
    #         condition = "process_log_ready";
    #       };
    #       mailhog = {
    #         condition = "process_started";
    #       };
    #       minio = {
    #         condition = "process_started";
    #       };
    #       watch_css = {
    #         condition = "process_started";
    #       };
    #     };
    #   };
    # };
    watch_css.exec = "bun run build:watch:css";
  };

  # See full reference at https://devenv.sh/reference/options/
}
