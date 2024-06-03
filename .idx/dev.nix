# To learn more about how to use Nix to configure your environment
# see: https://developers.google.com/idx/guides/customize-idx-env

{ pkgs, inputs, ... }: {
  channel = "stable-23.11"; # "stable-23.11" or "unstable"
  # Use https://search.nixos.org/packages to  find packages
  packages = [
    pkgs.go
    pkgs.air
    pkgs.goreleaser
    pkgs.gopls
    pkgs.gotools
    pkgs.go-tools
    # templ.packages.${system}.templ
    pkgs.nodejs-18_x
    pkgs.bun
    pkgs.gnumake
    pkgs.curl
    pkgs.git
    pkgs.jq
    pkgs.wget
    pkgs.flyctl
    pkgs.nixpkgs-fmt
    pkgs.zstd
  ];
  # Sets environment variables in the workspace
  env = { };
  # search for the extension on https://open-vsx.org/ and use "publisher.id"
  idx.extensions = [
    "golang.go"
    "alexcvzz.vscode-sqlite"
    "esbenp.prettier-vscode"
    "mjmlio.vscode-mjml"
    "mrmlnc.vscode-scss"
    "sibiraj-s.vscode-scss-formatter"
  ];
  idx.workspace.onCreate = {
    npm-install = ''
      npm install
      npm install -g concurrently
      go install github.com/a-h/templ/cmd/templ@latest
    '';
  };
  # preview configuration, identical to monospace.json
  idx.previews = {
    enable = true;
    previews = {
      web = {
        # command = [ "go" "run" "./" "-addr" "localhost:$PORT" ];
        command = [ "npm" "run" "dev" "--" "--port" "$PORT" "--hostname" "0.0.0.0" ];
        manager = "web";
      };
    };
  };
}
