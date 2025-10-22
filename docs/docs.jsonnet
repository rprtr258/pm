local toc = {
  compose(xs): std.foldl(function(acc, x)
    local node(x) = {title: x.title, children: []};
    if std.type(x) != "object" then acc
    else if x.level == 0 then acc + [node(x)]
    else if x.level == 1 then (
      local n = std.length(acc);
      local last = acc[n-1];
      acc[:n-1] + [{title: last.title, children: last.children + [node(x)]}]
    )
    else if x.level == 2 then (
      local n = std.length(acc);
      local last = acc[n-1];
      local m = std.length(last.children);
      local lastlast = last.children[m-1];
      acc[:n-1] + [{
        title: last.title,
        children: last.children[:m-1] + [{
          title: lastlast.title,
          children: lastlast.children + [node(x)]
        }],
      }]
    )
    else acc
    , xs, []),
  h1(title): {title: title, level: 0},
  h2(title): {title: title, level: 1},
  h3(title): {title: title, level: 2},
  ul(xs): [],
  p(xs): [],
  code(code): [],
  codeblock_sh(code): [],
  a(text, href): [],
  a_external(text, href): [],
  icon(): [],
  process_state_diagram: [],
};

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
  ul(lines): std.join("\n", [self.li(line) for line in lines])+"\n", // TODO: move out li
  li(x): "- "+std.join("", x),
  img(src, alt): "![%(alt)s](%(src)s)" % {src: src, alt: alt},
  hr: "---",
};

local markdown_adapter = {
  render(doc): renderer_markdown.compose(doc(markdown_adapter)),
  h1(title): renderer_markdown.h1(title),
  h2(title): renderer_markdown.h2(title),
  h3(title): renderer_markdown.h3(title),
  p(xs): renderer_markdown.p(xs),
  b(s): renderer_markdown.bold(s),
  a(text, href): renderer_markdown.a(text, href), // TODO: local links should work
  a_external(text, href): renderer_markdown.a(text, href),
  code(code): renderer_markdown.code(code),
  codeblock_sh(code): renderer_markdown.codeblock("sh", code),
  ul(xs): renderer_markdown.ul(xs),
  icon(): |||
    <p align="center"><img src="docs/icon.svg" width="250" height="250"></p>
  |||,
  process_state_diagram: |||
    ```mermaid
    flowchart TB
      0( )
      S(Stopped)
      C(Created)
      R(Running)
      A{{autorestart/watch enabled?}}
      0 -->|new process| S
      subgraph Running
        direction TB
        C -->|process started| R
        R -->|process died| A
      end
      A -->|yes| C
      A -->|no| S
      Running  -->|stop| S
      S -->|start| C
    ```
  |||,
};

local html_adapter = (
  local join(sep, fmt, o) = std.join(sep, [fmt % {k: k, v: o[k]} for k in std.objectFields(o)]);
  local renderCSSProps(o) = join(" ", "%(k)s: %(v)s;", o);
  local renderCSS(o) = join(
    "\n", "%(k)s { %(v)s }",
    std.mapWithKey(function(_, props) renderCSSProps(props), o),
  );

  {
    render(doc): (
      local TOC = toc.compose(doc(toc))[0].children; // NOTE: skip h1
      "<!DOCTYPE html>"+std.manifestXmlJsonml(["html", {lang: "en"},
        ["head", {},
          ["title", {}, "pm"],
          ["meta", {"http-equiv": "Content-Type", charset: "UTF-8"}],
          ["meta", {name: "description", content: "process manager"}],
          ["meta", {"http-equiv": "X-UA-Compatible", content: "IE=edge,chrome=1"}],
          ["meta", {name: "viewport", content: "width=device-width, initial-scale=1"}],
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
              ["div", {class: "sidebar-nav", role: "navigation", "aria-label": "primary"}, (
                local a(id, title) = ["a", {class: "section-link", href: "#"+id, title: title}, title];
                local toc_render(xs) = self.ul(std.foldl(function(acc, x) acc + [[a(x.title, x.title)], [toc_render(x.children)]], xs, []));
                toc_render(TOC)
              )],
            ],
            ["section", {class: "content"},
              ["article", {id: "main", class: "markdown-section", role: "main"}] + doc(html_adapter)]]]])
    ),
    process_state_diagram: import "process-state-diagram.jsonnet", // TODO: render from mermaid
    code(s): (
      local escape(s) = std.join("", std.map(function(c)
        if c == "<" then "&lt;"
        else if c == ">" then "&gt;"
        else c,
      s));
      ["code", {}, escape(s)]
    ),
    b(s): ["b", {}, s],
    span(s): ["span", {}, s],
    p(xs): ["p", {}] + xs,
    li(xs): ["li", {}] + xs,
    ul(xs): ["ul", {}] + [self.li(x) for x in xs],
    a(text, href): ["a", {href: "https://github.com/rprtr258/pm/blob/master/" + href}, text],
    a_external(text, href): ["a", {href: href, target: "_top"}, text],
    h1(title): ["h1", {id: title}, ["a", {href: "#"+title, class: "anchor"}, self.span(title)]],
    h2(title): ["h2", {id: title}, ["a", {href: "#"+title, class: "anchor"}, self.span(title)]],
    h3(title): ["h3", {id: title}, ["a", {href: "#"+title, class: "anchor"}, self.span(title)]],
    icon(): ["p", {align: "center"}, ["img", {src: "icon.svg", width: 250, height: 250, style: renderCSSProps({border: "0"})}]],
    img(src, width, height): ["img", {src: src, width: width, height: height, style: renderCSSProps({border: "0"})}],
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
  }
);

local link_release = "https://github.com/rprtr258/pm/releases/latest";

local docs(R) = [
  R.h1("PM (process manager)"),

  // ["div", {}, R.a("https://github.com/rprtr258/pm", R.img("https://img.shields.io/badge/source-code?logo=github&label=github"))],
  R.icon(),
  R.h2("Installation"),
  R.p(["PM is available only for linux due to heavy usage of linux mechanisms. Go to the ", R.a_external("releases", link_release), " page to download the latest binary."]),
  R.codeblock_sh(|||
    # download binary
    wget %(link_release)s/download/pm_linux_amd64
    # make binary executable
    chmod +x pm_linux_amd64
    # move binary to $PATH, here just local
    mv pm_linux_amd64 pm
  ||| % {link_release: link_release}),
    R.h3("Systemd service"),
    R.p(["To enable running processes on system startup:"]),
    R.codeblock_sh(|||
      # soft link /usr/bin/pm binary to whenever it is installed
      sudo ln -s ~/go/bin/pm /usr/bin/pm
      # install systemd service, copy/paste output of following command
      pm startup
    |||),
    R.p(["After these commands, processes with ", R.code("startup: true"), " config option will be started on system startup."]),

  R.h2("Configuration"),
  R.p([R.a_external("jsonnet", "https://jsonnet.org/"), " configuration language is used. It is also fully compatible with plain JSON, so you can write JSON instead."]),
  R.p(["See ", R.a("example configuration file", "./config.jsonnet"), ". Other examples can be found in ", R.a("tests", "./e2e/tests"), " directory."]),

  R.h2("Usage"),
  R.p(["Most fresh usage descriptions can be seen using ", R.code("pm <command> --help"), "."]),
    R.h3("Run process"),
    R.codeblock_sh(|||
      # run process using command
      pm run go run main.go

      # run processes from config file
      pm run --config config.jsonnet
    |||),
    R.h3("List processes"),
    R.codeblock_sh(|||
      pm list
    |||),

    R.h3("Start already added processes"),
    R.codeblock_sh(|||
      pm start [ID/NAME/TAG]...
    |||),

    R.h3("Stop processes"),
    R.codeblock_sh(|||
      pm stop [ID/NAME/TAG]...

      # e.g. stop all added processes (all processes has tag `all` by default)
      pm stop all
    |||),
    R.h3("Delete processes"),
    R.p(["When deleting process, they are first stopped, then removed from ", R.code("pm"), "."]),
    R.codeblock_sh(|||
      pm delete [ID/NAME/TAG]...

      # e.g. delete all processes
      pm delete all
    |||),

  R.h2("Process state diagram"),
  R.process_state_diagram,

  R.h2("Development"),
    R.h3("Architecture"),
    R.p([R.code("pm"), " consists of two parts:"]),
    R.ul([
      [R.b("cli client"), " - requests server, launches/stops shim processes"],
      [R.b("shim"), " - monitors and restarts processes, handle watches, signals and shutdowns"],
    ]),

    R.h3("PM directory structure"),
    R.p([
      R.code("pm"),
      " uses ",
      R.a("XDG", "https://specifications.freedesktop.org/basedir-spec/latest/"),
      " specification, so db and logs are in ",
      R.code("~/.local/share/pm"),
      " and config is ",
      R.code("~/.config/pm.json"),
      ". ",
      R.code("XDG_DATA_HOME"), " and ", R.code("XDG_CONFIG_HOME"),
      " environment variables can be used to change this. Layout is following:"]),
    R.codeblock_sh(|||
      ~/.config/pm.json # pm config file
      ~/.local/share/pm/
      ├──db/ # database tables
      │   └──<ID> # process info
      └──logs/ # processes logs
          ├──<ID>.stdout # stdout of process with id ID
          └──<ID>.stderr # stderr of process with id ID
    |||),

    R.h3("Differences from pm2"),
    R.ul([
      [R.code("pm"), " is just a single binary, not dependent on ", R.code("nodejs"), " and bunch of ", R.code("js"), " scripts"],
      [R.a_external("jsonnet", "https://jsonnet.org/"), " configuration language, back compatible with ", R.code("JSON"), " and allows to thoroughly configure processes, e.g. separate environments without requiring corresponding mechanism in ", R.code("pm"), " (others configuration languages might be added in future such as ", R.code("Procfile"), ", ", R.code("HCL"), ", etc.)"],
      ["supports only ", R.code("linux"), " now"],
      ["I can fix problems/add features as I need, independent of whether they work or not in ", R.code("pm2"), " because I don't know ", R.code("js")],
      ["fast and convenient (I hope so)"],
      ["no specific integrations for ", R.code("js")],
    ]),

    R.h3("Release"),
    R.p(["On ", R.code("master"), " branch:"]),
    R.codeblock_sh(|||
      git tag v1.2.3
      git push --tags
      goreleaser release --clean
    |||),
];

{
  "index.html": html_adapter.render(docs),
  "../readme.md": markdown_adapter.render(docs),
}
