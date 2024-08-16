// utils
local items(o) = [{k: k, v: o[k]} for k in std.objectFields(o)];
local join(sep, fmt, o) = std.join(sep, [fmt % kv for kv in items(o)]);
local renderCSSProps(o) = join(" ", "%(k)s: %(v)s;", o);
local renderCSS(o) = join(
  "\n", "%(k)s { %(v)s }",
  std.mapWithKey(function(_, props) renderCSSProps(props), o),
);

// TODO:
local renderer_markdown = {
  compose(xs): std.join("\n", xs),
  unknown(x): error "unknown renderer_markdown element: %(x)s" % {x: x},
  h1(title): "# "+title,
  h2(title): "## "+title,
  h3(title): "### "+title,
  p(xs): std.join("", xs)+"\n",
  code(code): "`"+code+"`",
  codeblock(lang, code): |||
    ```%(lang)s
    %(code)s```
  ||| % {lang: lang, code: code},
  a(text, href): "[%(text)s](%(href)s)" % {text: text, href: href},
  bold(text): "**%(text)s**" % {text: text},
  italic(text): "_%(text)s_" % {text: text},
  ul(xs): std.join("", ["\n- "+x for x in xs])+"\n",
  li(x): "- "+x,
  img(src, alt): "![%(alt)s](%(src)s)" % {src: src, alt: alt},
  hr: "---",
};
local link_release = "https://github.com/rprtr258/pm/releases/latest";
local content_example(R) = (
  local a_external = R.a;
  R.compose([
    R.h1("PM (process manager)"),
    R.h2("Installation"),
    R.p([
      "PM is available only for linux due to heavy usage of linux mechanisms. Go to the ",
      a_external("releases", link_release),
      " page to download the latest binary.",
    ]),
    R.codeblock("sh", |||
      # download binary
      wget %(link_release)s/download/pm_linux_amd64
      # make binary executable
      chmod +x pm_linux_amd64
      # move binary to $PATH, here just local
      mv pm_linux_amd64 pm
    ||| % {link_release: link_release}),
    R.h3("Systemd service"),
    R.p([
      "To enable running processes on system startup:",
      R.ul([
        "Copy "+R.a("pm.service", "./pm.service")+" file locally. This is the systemd service file that tells systemd how to manage your application.",
        "Change `User` field to your own username. This specifies under which user account the service will run, which affects permissions and environment.",
        "Change `ExecStart` to use `pm` binary installed. This is the command that systemd will execute to start your service.",
        "Move the file to `/etc/systemd/system/pm.service` and set root permissions on it:",
      ]),
      R.codeblock("sh", |||
        # copy service file to system's directory for systemd services
        sudo cp pm.service /etc/systemd/system/pm.service
        # set permission of service file to be readable and writable by owner, and readable by others
        sudo chmod 644 /etc/systemd/system/pm.service
        # change owner and group of service file to root, ensuring that it is managed by system administrator
        sudo chown root:root /etc/systemd/system/pm.service
        # reload systemd manager configuration, scanning for new or changed units
        sudo systemctl daemon-reload
        # enables service to start at boot time
        sudo systemctl enable pm
        # starts service immediately
        sudo systemctl start pm
        # soft link /usr/bin/pm binary to whenever it is installed
        sudo ln -s ~/go/bin/pm /usr/bin/pm
      |||),
      "After these commands, processes with "+R.code("startup: true")+" config option will be started on system startup."
    ]),
    R.h2("Configuration"),
    R.a("jsonnet", "https://jsonnet.org/") + " configuration language is used. It is also fully compatible with plain JSON, so you can write JSON instead.",
    "",
    "See "+R.a("example configuration file", "./config.jsonnet") + ". Other examples can be found in " + R.a("tests", "./tests") + " directory.",
    R.h2("Usage"),
    R.h3("Run process"),
    R.h3("List processes"),
    R.h3("Start already added processes"),
    R.h3("Stop processes"),
    R.h3("Delete processes"),
    R.h2("Process state diagram"),
    R.h2("Development"),
    R.h3("Architecture"),
    R.h3("PM directory structure"),
    R.h3("Differences from pm2"),
    R.h3("Release"),
  ])
);

local html_renderer = {
  code(s): ["code", {}, s],
  span(s): ["span", {}, s],
  p(xs): ["p", {}] + xs,
  li(xs): ["li", {}] + xs,
  ul(xs): ["ul", {}] + xs,
  ul_flat(xs): self.ul([self.li(x) for x in xs]),
  a(href, text): ["a", {href: href}, text],
  a_external(text, href): ["a", {href: href, target: "_top"}, text],
  h1(id, title): ["h1", {id: id}, ["a", {href: "#"+id, class: "anchor"}, self.span(title)]],
  h2(id, title): ["h2", {id: id}, ["a", {href: "#"+id, class: "anchor"}, self.span(title)]],
  h3(id, title): ["h3", {id: id}, ["a", {href: "#"+id, class: "anchor"}, self.span(title)]],
  img(src): ["img", {src: src, style: renderCSSProps({"max-width": "100%", border: "0"})}],
  codeblock_sh(code): (
    local functionn(s) = ["span", {style: renderCSSProps({color: "var(--code-theme-function)"})}, s];
    local variable(s) = ["span", {style: renderCSSProps({color: "var(--code-theme-variable)"})}, s];
    local comment(s) = ["span", {style: renderCSSProps({color: "var(--code-theme-comment)"})}, s];
    local operator(s) = ["span", {style: renderCSSProps({color: "var(--code-theme-operator)"})}, s];
    local env(s) = ["span", {style: renderCSSProps({color: "var(--code-theme-tag)"})}, s];
    local number(s) = ["span", {style: renderCSSProps({color: "var(--code-theme-tag)"})}, s];
    local punctuation(s) = ["span", {style: renderCSSProps({color: "var(--code-theme-punctuation)"})}, s];
    local render_word(word) =
      local functionns = ["wget", "chmod", "chown", "mv", "cp", "ln", "sudo", "git"];
      local lt = std.findSubstr("<", word);
      local gt = std.findSubstr(">", word);
      local lsb = std.findSubstr("[", word);
      local rsb = std.findSubstr("]", word);
      local ellipsis = std.findSubstr("...", word);
      if word == "" then []
      else if std.length(lt) > 0 then render_word(std.substr(word, 0, lt[0])) + [operator("&lt;")] + render_word(std.substr(word, lt[0]+1, std.length(word)))
      else if std.length(gt) > 0 then render_word(std.substr(word, 0, gt[0])) + [operator("&gt;")] + render_word(std.substr(word, gt[0]+1, std.length(word)))
      else if std.length(lsb) > 0 then render_word(std.substr(word, 0, lsb[0])) + [punctuation("[")] + render_word(std.substr(word, lsb[0]+1, std.length(word)))
      else if std.length(rsb) > 0 then render_word(std.substr(word, 0, rsb[0])) + [punctuation("]")] + render_word(std.substr(word, rsb[0]+1, std.length(word)))
      else if std.length(ellipsis) > 0 then render_word(std.substr(word, 0, ellipsis[0])) + [punctuation("...")] + render_word(std.substr(word, ellipsis[0]+3, std.length(word)))
      else if std.any([word == x for x in functionns]) then [functionn(word)]
      else if word == "[" then [punctuation(word)]
      else if word == "]" then [punctuation(word)]
      else if word == "644" then [number(word)]
      else if word == "enable" then [["span", {class: "token class-name", style: renderCSSProps({color: "var(--code-theme-selector)"})}, "enable"]]
      else if word[0] == "-" then [variable(word)]
      else if word == "$HOME/.pm/" then [env("$HOME"), "/.pm/"]
      else [word];
    local render(line) = // TODO: use sh parser actually
      if std.length(line) > 0 then
        local hash = std.findSubstr("#", line);
        local words = std.split(line, " ");
        if std.length(hash) > 0 then
          local before = std.substr(line, 0, hash[0]);
          local after = std.substr(line, hash[0], std.length(line));
          render(before) + [comment(after)]
        else std.join([" "], [
          render_word(word)
          for word in words
        ])
      else [line];
    local lines = [render(line) for line in std.split(code, "\n")];
    ["pre", {"data-lang": "sh", style: renderCSSProps({background: "var(--code-theme-background)"})},
      ["code", {class: "language-sh"}] + std.join(["\n"], lines)]
    ),
};
local R = html_renderer;

// docs
local dom = ["html", {lang: "en"},
  ["head", {},
    ["meta", {"http-equiv": "Content-Type", charset: "UTF-8"}],
    ["title", {}, "pm"],
    ["meta", {name: "description", content: "process manager"}],
    ["meta", {"http-equiv": "X-UA-Compatible", content: "IE=edge,chrome=1"}],
    ["meta", {"name": "viewport", content: "width=device-width, initial-scale=1"}],
    ["link", {rel: "icon", href: "assets/favicon/favicon.png"}],

    ["meta", {property: "og:title", content: "pm"}],
    ["meta", {property: "og:description", content: "process manager"}],
    ["meta", {property: "og:type", content: "website"}],
    ["meta", {property: "og:url", content: "https://rprtr258.github.io/pm/"}],
    ["meta", {property: "og:image", content: "https://rprtr258.github.io/pm/images/og-image.png"}],

    ["style", {}, renderCSS(import "styles.jsonnet")],
  ],
  ["body", {class: "sticky", style: renderCSSProps({margin: "0"})},
    ["main", {role: "presentation"},
      ["aside", {class: "sidebar", role: "none"},
        ["div", {class: "sidebar-nav", role: "navigation", "aria-label": "primary"},
          local a(id, title) = ["a", {class: "section-link", href: "#"+id, title: title}, title];
          R.ul_flat([
            [a("installation", "Installation")],
            [R.ul_flat([
              [a("systemd-service", "Systemd service")],
            ])],
            [a("configuration", "Configuration")],
            [a("usage", "Usage")],
            [R.ul_flat([
              [a("run-process", "Run process")],
              [a("list-processes", "List processes")],
              [a("start-already-added-processes", "Start already added processes")],
              [a("stop-processes", "Stop processes")],
              [a("delete-processes", "Delete processes")],
            ])],
            [a("process-state-diagram", "Process state diagram")],
            [a("development", "Development")],
            [R.ul_flat([
              [a("architecture", "Architecture")],
              [a("pm-directory-structure", "PM directory structure")],
              [a("differences-from-pm2", "Differences from pm2")],
              [a("release", "Release")],
            ])],
          ]),
        ],
      ],
      ["section", {class: "content"},
        ["article", {id: "main", class: "markdown-section", role: "main"},
          R.h1("pm-process-manager", "PM (process manager)"),

          ["div", {}, R.a("https://github.com/rprtr258/pm", R.img("https://img.shields.io/badge/source-code?logo=github&label=github"))],
          R.h2("installation", "Installation"),
            R.p(["PM is available only for linux due to heavy usage of linux mechanisms. Go to the ", R.a_external("releases", link_release), " page to download the latest binary."]),
            R.codeblock_sh(|||
              # download binary
              wget %(link_release)s/download/pm_linux_amd64
              # make binary executable
              chmod +x pm_linux_amd64
              # move binary to $PATH, here just local
              mv pm_linux_amd64 pm
            ||| % {link_release: link_release}),
            R.h3("systemd-service", "Systemd service"),
              R.p(["To enable running processes on system startup:"]),
              R.ul_flat([
                ["Copy", R.a("#/pm.service", R.code("pm.service")), "file locally. This is the systemd service file that tells systemd how to manage your application."],
                ["Change", R.code("User"), "field to your own username. This specifies under which user account the service will run, which affects permissions and environment."],
                ["Change", R.code("ExecStart"), "to use", R.code("pm"), "binary installed. This is the command that systemd will execute to start your service."],
                ["Move the file to", R.code("/etc/systemd/system/pm.service"), "and set root permissions on it:"],
              ]),
              R.codeblock_sh(|||
                # copy service file to system's directory for systemd services
                sudo cp pm.service /etc/systemd/system/pm.service
                # set permission of service file to be readable and writable by owner, and readable by others
                sudo chmod 644 /etc/systemd/system/pm.service
                # change owner and group of service file to root, ensuring that it is managed by system administrator
                sudo chown root:root /etc/systemd/system/pm.service
                # reload systemd manager configuration, scanning for new or changed units
                sudo systemctl daemon-reload
                # enables service to start at boot time
                sudo systemctl enable pm
                # starts service immediately
                sudo systemctl start pm
                # soft link /usr/bin/pm binary to whenever it is installed
                sudo ln -s ~/go/bin/pm /usr/bin/pm
              |||),
              R.p(["After these commands, processes with", R.code("startup: true"), "config option will be started on system startup."]),

          R.h2("configuration", "Configuration"),
            R.p([R.a_external("jsonnet", "https://jsonnet.org/"), " configuration language is used. It is also fully compatible with plain JSON, so you can write JSON instead."]),
            R.p(["See ", R.a("#/config.jsonnet", "example configuration file"), ". Other examples can be found in ", R.a("#/tests", "tests"), " directory."]),

          R.h2("usage", "Usage"),
            R.p(["Most fresh usage descriptions can be seen using", R.code("pm &lt;command&gt; --help"), "."]),
            R.h3("run-process", "Run process"),
              R.codeblock_sh(|||
                # run process using command
                pm run go run main.go

                # run processes from config file
                pm run --config config.jsonnet
              |||),
            R.h3("list-processes", "List processes"),
              R.codeblock_sh(|||
                pm list
              |||),

            R.h3("start-already-added-processes", "Start already added processes"),
              R.codeblock_sh(|||
                pm start [ID/NAME/TAG]...
              |||),

            R.h3("stop-processes", "Stop processes"),
              R.codeblock_sh(|||
                pm stop [ID/NAME/TAG]...

                # e.g. stop all added processes (all processes has tag `all` by default)
                pm stop all
              |||),
            R.h3("delete-processes", "Delete processes"),
              R.p(["When deleting process, they are first stopped, then removed from", R.code("pm"), "."]),
              R.codeblock_sh(|||
                pm delete [ID/NAME/TAG]...

                # e.g. delete all processes
                pm delete all
              |||),

          R.h2("process-state-diagram", "Process state diagram"),
            import "process-state-diagram.jsonnet",

          R.h2("development", "Development"),
            R.h3("architecture", "Architecture"),
              R.p([R.code("pm"), "consists of two parts:"]),
              local b = function(x) ["b", {}, x];
              R.ul_flat([
                [b("cli client"), " - requests server, launches/stops shim processes"],
                [b("shim"), " - monitors and restarts processes, handle watches, signals and shutdowns"],
              ]),

            R.h3("pm-directory-structure", "PM directory structure"),
              R.p([R.code("pm"), "uses directory", R.code("$HOME/.pm"), "to store data by default.", R.code("PM_HOME"), "environment variable can be used to change this. Layout is following:"]),
              R.codeblock_sh(|||
                $HOME/.pm/
                ├──config.json # pm config file
                ├──db/ # database tables
                │   └──<ID> # process info
                └──logs/ # processes logs
                   ├──<ID>.stdout # stdout of process with id ID
                   └──<ID>.stderr # stderr of process with id ID
              |||),

            R.h3("differences-from-pm2", "Differences from pm2"),
              R.ul_flat([
                [R.code("pm"), "is just a single binary, not dependent on", R.code("nodejs"), "and bunch of", R.code("js"), "scripts"],
                [R.a_external("jsonnet", "https://jsonnet.org/"), " configuration language, back compatible with", R.code("JSON"), "and allows to thoroughly configure processes, e.g. separate environments without requiring corresponding mechanism in", R.code("pm"), "(others configuration languages might be added in future such as", R.code("Procfile"), R.code("HCL"), "etc.)"],
                ["supports only", R.code("linux"), "now"],
                ["I can fix problems/add features as I need, independent of whether they work or not in", R.code("pm2"), "because I don’t know", R.code("js")],
                ["fast and convenient (I hope so)"],
                ["no specific integrations for", R.code("js")],
              ]),

            R.h3("release", "Release"),
              R.p(["On", R.code("master"), "branch:"]),
              R.codeblock_sh(|||
                git tag v1.2.3
                git push --tags
                goreleaser release --clean
              |||),
        ],
      ],
    ],
  ],
];

{
  "index.html": "<!DOCTYPE html>"+std.manifestXmlJsonml(dom),
  "readme.md": importstr "../readme.md",
  "example.md": content_example(renderer_markdown),
}
