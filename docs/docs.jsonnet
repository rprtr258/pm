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
  p(xs): xs+"\n",
  codeblock(lang, code): |||
    ```%(lang)s
    %(code)s
    ```
  ||| % {lang: lang, code: code},
  a(href, text): "[%(text)s](%(href)s)" % {text: text, href: href},
  bold(text): "**%(text)s**" % {text: text},
  italic(text): "_%(text)s_" % {text: text},
  ul(xs): std.join("\n", xs)+"\n",
  li(x): "- "+x,
  img(src, alt): "![%(alt)s](%(src)s)" % {src: src, alt: alt},
  hr: "---",
};
local content_example(renderer) = renderer.compose([
  renderer.h1("pm-process-manager", "PM (process manager)"),
  renderer.p(["PM is available only for linux due to heavy usage of linux mechanisms. Go to the ", renderer.a_external("https://github.com/rprtr258/pm/releases/latest", "releases"), " page to download the latest binary."]),
  renderer.codeblock_sh(function(h) [
    [h.comment("# download binary")],
    [h.functionn("wget"), " https://github.com/rprtr258/pm/releases/latest/download/pm_linux_amd64"],
    [h.comment("# make binary executable")],
    [h.functionn("chmod"), " +x pm_linux_amd64"],
    [h.comment("# move binary to $PATH, here just local")],
    [h.functionn("mv"), " pm_linux_amd64 pm"],
  ]),
]);

local code(s) = ["code", {}, s];
local span(s) = ["span", {}, s];
local p(xs) = ["p", {}] + xs;
local li(xs) = ["li", {}] + xs;
local ul(xs) = ["ul", {}] + xs;
local ul_flat(xs) = ul([li(x) for x in xs]);
local a(href, text) = ["a", {href: href}, text];
local a_external(href, text) = ["a", {href: href, target: "_top"}, text];

local h1(id, title) = ["h1", {id: id}, ["a", {href: "#"+id, class: "anchor"}, span(title)]];
local h2(id, title) = ["h2", {id: id}, ["a", {href: "#"+id, class: "anchor"}, span(title)]];
local h3(id, title) = ["h3", {id: id}, ["a", {href: "#"+id, class: "anchor"}, span(title)]];
local codeblock_sh(lines) = ["pre", {"data-lang": "sh", class: "language-sh"},
  ["code", {class: "lang-sh language-sh"}] + std.join(["\n"], lines({
    functionn(s): ["span", {class: "token function"}, s],
    variable(s): ["span", {class: "token parameter variable"}, s],
    comment(s): ["span", {class: "token comment"}, s],
    operator(s): ["span", {class: "token operator"}, s],
    env(s): ["span", {class: "token environment constant"}, s],
    punctuation(s): ["span", {class: "token punctuation"}, s],
  }))];

// docs
local dom = ["html", {lang: "en", class: "themeable", style: renderCSSProps({
  "--navbar-root-color--active":             "#0374B5",
  "--blockquote-border-color":               "#0374B5",
  "--sidebar-name-color":                    "#0374B5",
  "--sidebar-nav-link-color--active":        "#0374B5",
  "--sidebar-nav-link-border-color--active": "#0374B5",
  "--link-color":                            "#0374B5",
  "--pagination-title-color":                "#0374B5",
  "--cover-link-color":                      "#0374B5",
  "--cover-button-primary-color":            "#FFFFFF",
  "--cover-button-primary-background":       "#0374B5",
  "--cover-button-primary-border":           "1px solid #0374B5",
  "--cover-button-color":                    "#0374B5",
  "--cover-button-border":                   "1px solid #0374B5",
  "--cover-background-color":                "#6c8a9a",
  "--sidebar-nav-pagelink-background--active": "no-repeat 0px center / 5px 6px linear-gradient(225deg, transparent 2.75px, #0374B5 2.75px 4.25px, transparent 4.25px), no-repeat 5px center / 5px 6px linear-gradient(135deg, transparent 2.75px, #0374B5 2.75px 4.25px, transparent 4.25px)",
  "--sidebar-nav-pagelink-background--collapse": "no-repeat 2px calc(50% - 2.5px) / 6px 5px linear-gradient(45deg, transparent 2.75px, #0374B5 2.75px 4.25px, transparent 4px), no-repeat 2px calc(50% + 2.5px) / 6px 5px linear-gradient(135deg, transparent 2.75px, #0374B5 2.75px 4.25px, transparent 4px)",
  "--sidebar-nav-pagelink-background--loaded": "no-repeat 0px center / 5px 6px linear-gradient(225deg, transparent 2.75px, #0374B5 2.75px 4.25px, transparent 4.25px), no-repeat 5px center / 5px 6px linear-gradient(135deg, transparent 2.75px, #0374B5 2.75px 4.25px, transparent 4.25px)",
})},
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

    ["link", {rel: "stylesheet", href: "./styles.css"}],
  ],
  ["body", {class: "ready sticky ready-fix vsc-initialized"},
    ["main", {role: "presentation"},
      ["aside", {id: "__sidebar", class: "sidebar", role: "none"},
        ["div", {class: "sidebar-nav", role: "navigation", "aria-label": "primary"},
          local a(id, title) = ["a", {class: "section-link", href: "#"+id, title: title}, title];
          ul([
            li([a("installation", "Installation")]),
            ul_flat([
              [a("systemd-service", "Systemd service")],
            ]),
            li([a("configuration", "Configuration")]),
            li([a("usage", "Usage")]),
            ul_flat([
              [a("run-process", "Run process")],
              [a("list-processes", "List processes")],
              [a("start-processes-that-already-has-been-added", "Start processes that already has been added")],
              [a("stop-processes", "Stop processes")],
              [a("delete-processes", "Delete processes")],
            ]),
            li([a("process-state-diagram", "Process state diagram")]),
            li([a("development", "Development")]),
            ul_flat([
              [a("architecture", "Architecture")],
              [a("pm-directory-structure", "PM directory structure")],
              [a("differences-from-pm2", "Differences from pm2")],
              [a("release", "Release")],
            ]),
          ]),
        ],
      ],
      ["section", {class: "content"},
        ["article", {id: "main", class: "markdown-section", role: "main"},
          h1("pm-process-manager", "PM (process manager)"),

          ["div", {}, a("https://github.com/rprtr258/pm", ["img", {src: "https://img.shields.io/badge/source-code?logo=github&label=github"}])],
          h2("installation", "Installation"),
            p(["PM is available only for linux due to heavy usage of linux mechanisms. Go to the ", a_external("https://github.com/rprtr258/pm/releases/latest", "releases"), " page to download the latest binary."]),
            codeblock_sh(function(h) [
              [h.comment("# download binary")],
              [h.functionn("wget"), " https://github.com/rprtr258/pm/releases/latest/download/pm_linux_amd64"],
              [h.comment("# make binary executable")],
              [h.functionn("chmod"), " +x pm_linux_amd64"],
              [h.comment("# move binary to $PATH, here just local")],
              [h.functionn("mv"), " pm_linux_amd64 pm"],
            ]),
            h3("systemd-service", "Systemd service"),
              p(["To enable running processes on system startup:"]),
              ul_flat([
                ["Copy", a("#/pm.service", code("pm.service")), "file locally. This is the systemd service file that tells systemd how to manage your application."],
                ["Change", code("User"), "field to your own username. This specifies under which user account the service will run, which affects permissions and environment."],
                ["Change", code("ExecStart"), "to use", code("pm"), "binary installed. This is the command that systemd will execute to start your service."],
                ["Move the file to", code("/etc/systemd/system/pm.service"), "and set root permissions on it:"],
              ]),
              codeblock_sh(function(h) [
                [h.comment("# copy service file to system's directory for systemd services")],
                [h.functionn("sudo"), " ", h.functionn("cp"), " pm.service /etc/systemd/system/pm.service"],
                [h.comment("# set permission of service file to be readable and writable by owner, and readable by others")],
                [h.functionn("sudo"), " ", h.functionn("chmod"), " ", ["span", {class: "token number"}, "644"], "/etc/systemd/system/pm.service"],
                [h.comment("# change owner and group of service file to root, ensuring that it is managed by system administrator")],
                [h.functionn("sudo"), " ", h.functionn("chown"), " root:root /etc/systemd/system/pm.service"],
                [h.comment("# reload systemd manager configuration, scanning for new or changed units")],
                [h.functionn("sudo"), " systemctl daemon-reload"],
                [h.comment("# enables service to start at boot time")],
                [h.functionn("sudo"), " systemctl ", ["span", {class: "token builtin class-name"}, "enable"], " pm"],
                [h.comment("# starts service immediately")],
                [h.functionn("sudo"), " systemctl start pm"],
                [h.comment("# soft link /usr/bin/pm binary to whenever it is installed")],
                [h.functionn("sudo"), " ", h.functionn("ln"), " ", h.variable("-s"), " ~/go/bin/pm /usr/bin/pm"],
              ]),
              p(["After these commands, processes with", code("startup: true"), "config option will be started on system startup."]),

          h2("configuration", "Configuration"),
            p([a_external("https://jsonnet.org/", "jsonnet"), " configuration language is used. It is also fully compatible with plain JSON, so you can write JSON instead."]),
            p(["See ", a("#/config.jsonnet", "example configuration file"), ". Other examples can be found in ", a("#/tests", "tests"), " directory."]),

          h2("usage", "Usage"),
            p(["Most fresh usage descriptions can be seen using", code("pm &lt;command&gt; --help"), "."]),
            h3("run-process", "Run process"),
              codeblock_sh(function(h) [
                [h.comment("# run process using command")],
                ["pm run go run main.go"],
                [],
                [h.comment("# run processes from config file")],
                ["pm run ", h.variable("--config"), " config.jsonnet"],
              ]),
            h3("list-processes", "List processes"),
              codeblock_sh(function(h) [
                ["pm list"],
              ]),

            h3("start-processes-that-already-has-been-added", "Start processes that already has been added"),
              codeblock_sh(function(h) [
                ["pm start ", h.punctuation("["), "ID/NAME/TAG", h.punctuation("]"), h.punctuation("...")],
              ]),

            h3("stop-processes", "Stop processes"),
              codeblock_sh(function(h) [
                ["pm stop ", h.punctuation("["), "ID/NAME/TAG", h.punctuation("]"), h.punctuation("...")],
                [],
                [h.comment("# e.g. stop all added processes (all processes has tag `all` by default)")],
                ["pm stop all"],
              ]),
            h3("delete-processes", "Delete processes"),
              p(["When deleting process, they are first stopped, then removed from", code("pm"), "."]),
              codeblock_sh(function(h) [
                ["pm delete ", h.punctuation("["), "ID/NAME/TAG", h.punctuation("]"), h.punctuation("...")],
                [],
                [h.comment("# e.g. delete all processes")],
                ["pm delete all"],
              ]),

          h2("process-state-diagram", "Process state diagram"),
            import "process-state-diagram.jsonnet",

          h2("development", "Development"),
            h3("architecture", "Architecture"),
              p([code("pm"), "consists of two parts:"]),
              local b = function(x) ["b", {}, x];
              ul_flat([
                [b("cli client"), " - requests server, launches/stops shim processes"],
                [b("shim"), " - monitors and restarts processes, handle watches, signals and shutdowns"],
              ]),

            h3("pm-directory-structure", "PM directory structure"),
              p([code("pm"), "uses directory", code("$HOME/.pm"), "to store data by default.", code("PM_HOME"), "environment variable can be used to change this. Layout is following:"]),
              codeblock_sh(function(h) [
                [h.env("$HOME"), "/.pm/"],
                ["├──config.json ", h.comment("# pm config file")],
                ["├──db/ ", h.comment("# database tables")],
                ["│   └──", h.operator("&lt;"), "ID", h.operator("&gt;"), " ", h.comment("# process info")],
                ["└──logs/ ", h.comment("# processes logs")],
                ["   ├──", h.operator("&lt;"), "ID", h.operator("&gt;"), ".stdout ", h.comment("# stdout of process with id ID")],
                ["   └──", h.operator("&lt;"), "ID", h.operator("&gt;"), ".stderr ", h.comment("# stderr of process with id ID")],
              ]),

            h3("differences-from-pm2", "Differences from pm2"),
              ul_flat([
                [code("pm"), "is just a single binary, not dependent on", code("nodejs"), "and bunch of", code("js"), "scripts"],
                [a_external("https://jsonnet.org/", "jsonnet"), " configuration language, back compatible with", code("JSON"), "and allows to thoroughly configure processes, e.g. separate environments without requiring corresponding mechanism in", code("pm"), "(others configuration languages might be added in future such as", code("Procfile"), code("HCL"), "etc.)"],
                ["supports only", code("linux"), "now"],
                ["I can fix problems/add features as I need, independent of whether they work or not in", code("pm2"), "because I don’t know", code("js")],
                ["fast and convenient (I hope so)"],
                ["no specific integrations for", code("js")],
              ]),

            h3("release", "Release"),
              p(["On", code("master"), "branch:"]),
              codeblock_sh(function(h) [
                [h.functionn("git"), " tag v1.2.3"],
                [h.functionn("git"), " push ", h.variable("--tags")],
                [h.functionn("goreleaser"), " release ", h.variable("--clean")],
              ]),
        ],
      ],
    ],
  ],
];

{
  "index.html": "<!DOCTYPE html>"+std.manifestXmlJsonml(dom),
  "readme.md": importstr "../readme.md",
}
