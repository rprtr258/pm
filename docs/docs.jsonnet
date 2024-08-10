// utils
local items(o) = [{k: k, v: o[k]} for k in std.objectFields(o)];
local join(sep, fmt, o) = std.join(sep, [fmt % kv for kv in items(o)]);
local renderCSSProps(o) = join(" ", "%(k)s: %(v)s;", o);
local renderCSS(o) = join(
  "\n", "%(k)s { %(v)s }",
  std.mapWithKey(function(_, props) renderCSSProps(props), o),
);


// config
local repo = "rprtr258/pm";
local title = repo;
local description = "%(title)s is cool!" % {title: title};
local links = [
  // {"text": "Link 1", "link": "/link"},
];
// available themes as to https://docsify.js.org/#/themes
// vue, buble, dark, pure
local theme = "https://cdn.jsdelivr.net/npm/docsify@4/lib/themes/%(theme)s.css" % {theme: "buble"};
local config = {
  subMaxLevel: 1,
  maxLevel: 3,
  auto2top: true,
  repo: repo,
  routerMode: "history",
  relativePath: true,
  basePath: "https://rprtr258.github.io/pm/",
  homepage: "https://raw.githubusercontent.com/rprtr258/pm/master/readme.md",
};
local style = renderCSS({
  ".markdown-section": { "max-width": "90%" },
  "body": { "font-size": "13pt" },
});

// docs
local dom = ["html", {lang: "en"},
  ["head", {},
    ["meta", {charset: "UTF-8"}],
    ["title", {}, title],
    ["meta", {"http-equiv": "X-UA-Compatible", content: "IE=edge,chrome=1"}],
    ["meta", {name: "description", content: description}],
    ["meta", {name: "viewport", content: "width=device-width, user-scalable=no, initial-scale=1.0, maximum-scale=1.0, minimum-scale=1.0"}],
    ["link", {rel: "stylesheet", href: theme}],
    ["style", {}, style],
  ],
  ["body", {},
    ["nav", {style: renderCSSProps({
      display: "flex",
      "flex-direction": "row",
      "align-items": "center",
      "justify-content": "space-between",
    })},
      ["ul", {}] + [["li", {href: link.link}, link.text] for link in links],
    ],
    ["div", {id: "app"}],
    ["script", {}, |||
      window.$docsify = %(config)s;
    ||| % {config: std.manifestJson(config)}],
    ["script", {src: "https://cdn.jsdelivr.net/npm/docsify@4.12.2/lib/docsify.js"}],
    ["script", {src: "https://cdn.jsdelivr.net/npm/prismjs@1.28.0/prism.min.js"}],
    ["script", {src: "https://unpkg.com/docsify/lib/plugins/search.min.js"}],
    ["script", {type: "module"}, |||
      import mermaid from "https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.esm.min.mjs";
      mermaid.initialize({ startOnLoad: true });
      window.mermaid = mermaid;
    |||],
    ["script", {src: "https://unpkg.com/docsify-mermaid@2.0.1/dist/docsify-mermaid.js"}],
  ],
];

{
 //std.manifestXmlJsonml(dom),
  "index.html": "<!DOCTYPE html>"+|||
    <html lang="en" class="themeable" style="--cover-button-primary-color: #FFFFFF; --navbar-root-color--active: #0374B5; --blockquote-border-color: #0374B5; --sidebar-name-color: #0374B5; --sidebar-nav-link-color--active: #0374B5; --sidebar-nav-link-border-color--active: #0374B5; --link-color: #0374B5; --pagination-title-color: #0374B5; --cover-link-color: #0374B5; --cover-button-primary-background: #0374B5; --cover-button-primary-border: 1px solid #0374B5; --cover-button-color: #0374B5; --cover-button-border: 1px solid #0374B5; --sidebar-nav-pagelink-background--active: no-repeat 0px center / 5px 6px linear-gradient(225deg, transparent 2.75px, #0374B5 2.75px 4.25px, transparent 4.25px), no-repeat 5px center / 5px 6px linear-gradient(135deg, transparent 2.75px, #0374B5 2.75px 4.25px, transparent 4.25px); --sidebar-nav-pagelink-background--collapse: no-repeat 2px calc(50% - 2.5px) / 6px 5px linear-gradient(45deg, transparent 2.75px, #0374B5 2.75px 4.25px, transparent 4px), no-repeat 2px calc(50% + 2.5px) / 6px 5px linear-gradient(135deg, transparent 2.75px, #0374B5 2.75px 4.25px, transparent 4px); --sidebar-nav-pagelink-background--loaded: no-repeat 0px center / 5px 6px linear-gradient(225deg, transparent 2.75px, #0374B5 2.75px 4.25px, transparent 4.25px), no-repeat 5px center / 5px 6px linear-gradient(135deg, transparent 2.75px, #0374B5 2.75px 4.25px, transparent 4.25px); --cover-background-color: #6c8a9a;">
    <head>
      <meta http-equiv="Content-Type" charset="UTF-8">
      <title>pm</title>
      <meta name="description" content="process manager">
      <meta http-equiv="X-UA-Compatible" content="IE=edge,chrome=1">
      <meta name="viewport" content="width=device-width, initial-scale=1">
      <link rel="icon" href="assets/favicon/favicon.png">

      <meta property="og:title" content="pm">
      <meta property="og:description" content="process manager">
      <meta property="og:type" content="website">
      <meta property="og:url" content="https://rprtr258.github.io/pm/">
      <meta property="og:image" content="https://rprtr258.github.io/pm/images/og-image.png">

      <link rel="stylesheet" href="./styles.css">
    </head>

    <body class="ready sticky ready-fix vsc-initialized"><main role="presentation">
      <aside id="__sidebar" class="sidebar" role="none">
        <div class="sidebar-nav" role="navigation" aria-label="primary">
          <ul>
            <li><a class="section-link" href="#installation" title="Installation">Installation</a></li>
            <ul>
              <li><a class="section-link" href="#systemd-service" title="Systemd service">Systemd service</a></li>
            </ul>
            <li><a class="section-link" href="#configuration" title="Configuration">Configuration</a></li>
            <li><a class="section-link" href="#usage" title="Usage">Usage</a></li>
            <ul>
              <li><a class="section-link" href="#run-process" title="Run process">Run process</a></li>
              <li><a class="section-link" href="#list-processes" title="List processes">List processes</a></li>
              <li><a class="section-link" href="#start-processes-that-already-has-been-added" title="Start processes that already has been added">Start processes that already has been added</a></li>
              <li><a class="section-link" href="#stop-processes" title="Stop processes">Stop processes</a></li>
              <li><a class="section-link" href="#delete-processes" title="Delete processes">Delete processes</a></li>
            </ul>
            <li><a class="section-link" href="#process-state-diagram" title="Process state diagram">Process state diagram</a></li>
            <li><a class="section-link" href="#development" title="Development">Development</a></li>
            <ul>
              <li><a class="section-link" href="#architecture" title="Architecture">Architecture</a></li>
              <li><a class="section-link" href="#pm-directory-structure" title="PM directory structure">PM directory structure</a></li>
              <li><a class="section-link" href="#differences-from-pm2" title="Differences from pm2">Differences from pm2</a></li>
              <li><a class="section-link" href="#release" title="Release">Release</a></li>
            </ul>
          </ul>
        </div>
      </aside>
      <section class="content">
        <article id="main" class="markdown-section" role="main" tabindex="-1">
          <h1 id="pm-process-manager" tabindex="-1"><a href="#pm-process-manager" class="anchor"><span>PM (process manager)</span></a></h1>
          <div><a href="https://github.com/rprtr258/pm"><img src="https://img.shields.io/badge/source-code?logo=github&label=github"></a></div>
          <h2 id="installation" tabindex="-1"><a href="#installation" class="anchor"><span>Installation</span></a></h2>
          <p>PM is available only for linux due to heavy usage of linux mechanisms. Go to the <a href="https://github.com/rprtr258/pm/releases/latest" target="_top">releases</a> page to download the latest binary.</p>
          <pre data-lang="sh" class="language-sh"><code class="lang-sh language-sh" tabindex="0"><span class="token comment"># download binary</span>
    <span class="token function">wget</span> https://github.com/rprtr258/pm/releases/latest/download/pm_linux_amd64
    <span class="token comment"># make binary executable</span>
    <span class="token function">chmod</span> +x pm_linux_amd64
    <span class="token comment"># move binary to $PATH, here just local</span>
    <span class="token function">mv</span> pm_linux_amd64 pm</code></pre>

            <h3 id="systemd-service" tabindex="-1"><a href="#systemd-service" class="anchor"><span>Systemd service</span></a></h3>
            <p>To enable running processes on system startup:</p>
            <ul>
              <li>Copy <a href="#/pm.service"><code>pm.service</code></a> file locally. This is the systemd service file that tells systemd how to manage your application.</li>
              <li>Change <code>User</code> field to your own username. This specifies under which user account the service will run, which affects permissions and environment.</li>
              <li>Change <code>ExecStart</code> to use <code>pm</code> binary installed. This is the command that systemd will execute to start your service.</li>
              <li>Move the file to <code>/etc/systemd/system/pm.service</code> and set root permissions on it:</li>
            </ul>
            <pre data-lang="sh" class="language-sh"><code class="lang-sh language-sh" tabindex="0"><span class="token comment"># copy service file to system's directory for systemd services</span>
    <span class="token function">sudo</span> <span class="token function">cp</span> pm.service /etc/systemd/system/pm.service
    <span class="token comment"># set permission of service file to be readable and writable by owner, and readable by others</span>
    <span class="token function">sudo</span> <span class="token function">chmod</span> <span class="token number">644</span> /etc/systemd/system/pm.service
    <span class="token comment"># change owner and group of service file to root, ensuring that it is managed by system administrator</span>
    <span class="token function">sudo</span> <span class="token function">chown</span> root:root /etc/systemd/system/pm.service
    <span class="token comment"># reload systemd manager configuration, scanning for new or changed units</span>
    <span class="token function">sudo</span> systemctl daemon-reload
    <span class="token comment"># enables service to start at boot time</span>
    <span class="token function">sudo</span> systemctl <span class="token builtin class-name">enable</span> pm
    <span class="token comment"># starts service immediately</span>
    <span class="token function">sudo</span> systemctl start pm
    <span class="token comment"># soft link /usr/bin/pm binary to whenever it is installed</span>
    <span class="token function">sudo</span> <span class="token function">ln</span> <span class="token parameter variable">-s</span> ~/go/bin/pm /usr/bin/pm</code></pre>
            <p>After these commands, processes with <code>startup: true</code> config option will be started on system startup.</p>

            <h2 id="configuration" tabindex="-1"><a href="#configuration" class="anchor"><span>Configuration</span></a></h2>
            <p><a href="https://jsonnet.org/" target="_top">jsonnet</a> configuration language is used. It is also fully compatible with plain JSON, so you can write JSON instead.</p>
            <p>See <a href="#/config.jsonnet">example configuration file</a>. Other examples can be found in <a href="#/tests">tests</a> directory.</p>

          <h2 id="usage" tabindex="-1"><a href="#usage" data-id="usage" class="anchor"><span>Usage</span></a></h2><p>Most fresh usage descriptions can be seen using <code>pm &lt;command&gt; --help</code>.</p>
            <h3 id="run-process" tabindex="-1"><a href="#run-process" data-id="run-process" class="anchor"><span>Run process</span></a></h3>
            <pre data-lang="sh" class="language-sh"><code class="lang-sh language-sh" tabindex="0"><span class="token comment"># run process using command</span>
    pm run go run main.go

    <span class="token comment"># run processes from config file</span>
    pm run <span class="token parameter variable">--config</span> config.jsonnet</code></pre>
            <h3 id="list-processes" tabindex="-1"><a href="#list-processes" data-id="list-processes" class="anchor"><span>List processes</span></a></h3>
            <pre data-lang="sh" class="language-sh"><code class="lang-sh language-sh" tabindex="0">pm list</code></pre>

            <h3 id="start-processes-that-already-has-been-added" tabindex="-1"><a href="#start-processes-that-already-has-been-added" data-id="start-processes-that-already-has-been-added" class="anchor"><span>Start processes that already has been added</span></a></h3>
            <pre data-lang="sh" class="language-sh"><code class="lang-sh language-sh" tabindex="0">pm start <span class="token punctuation">[</span>ID/NAME/TAG<span class="token punctuation">]</span><span class="token punctuation">..</span>.</code></pre>

            <h3 id="stop-processes" tabindex="-1"><a href="#stop-processes" data-id="stop-processes" class="anchor"><span>Stop processes</span></a></h3>
            <pre data-lang="sh" class="language-sh"><code class="lang-sh language-sh" tabindex="0">pm stop <span class="token punctuation">[</span>ID/NAME/TAG<span class="token punctuation">]</span><span class="token punctuation">..</span>.
    <span class="token comment"># e.g. stop all added processes (all processes has tag `all` by default)</span>
    pm stop all</code></pre><h3 id="delete-processes" tabindex="-1"><a href="#delete-processes" data-id="delete-processes" class="anchor"><span>Delete processes</span></a></h3><p>When deleting process, they are first stopped, then removed from <code>pm</code>.</p><pre data-lang="sh" class="language-sh"><code class="lang-sh language-sh" tabindex="0">pm delete <span class="token punctuation">[</span>ID/NAME/TAG<span class="token punctuation">]</span><span class="token punctuation">..</span>.
    <span class="token comment"># e.g. delete all processes</span>
    pm delete all</code></pre>
            <h2 id="process-state-diagram" tabindex="-1"><a href="#process-state-diagram" data-id="process-state-diagram" class="anchor"><span>Process state diagram</span></a></h2>
            <div class="mermaid" data-processed="true">
              <svg aria-roledescription="flowchart-v2" role="graphics-document document" viewBox="-8 -8 370.84375 520.125" style="max-width: 370.84375px;" xmlns:xlink="http://www.w3.org/1999/xlink" xmlns="http://www.w3.org/2000/svg" width="100%" id="mermaid-1722776679139"><style>#mermaid-1722776679139{font-family:"trebuchet ms",verdana,arial,sans-serif;font-size:16px;fill:#333;}#mermaid-1722776679139 .error-icon{fill:#552222;}#mermaid-1722776679139 .error-text{fill:#552222;stroke:#552222;}#mermaid-1722776679139 .edge-thickness-normal{stroke-width:2px;}#mermaid-1722776679139 .edge-thickness-thick{stroke-width:3.5px;}#mermaid-1722776679139 .edge-pattern-solid{stroke-dasharray:0;}#mermaid-1722776679139 .edge-pattern-dashed{stroke-dasharray:3;}#mermaid-1722776679139 .edge-pattern-dotted{stroke-dasharray:2;}#mermaid-1722776679139 .marker{fill:#333333;stroke:#333333;}#mermaid-1722776679139 .marker.cross{stroke:#333333;}#mermaid-1722776679139 svg{font-family:"trebuchet ms",verdana,arial,sans-serif;font-size:16px;}#mermaid-1722776679139 .label{font-family:"trebuchet ms",verdana,arial,sans-serif;color:#333;}#mermaid-1722776679139 .cluster-label text{fill:#333;}#mermaid-1722776679139 .cluster-label span,#mermaid-1722776679139 p{color:#333;}#mermaid-1722776679139 .label text,#mermaid-1722776679139 span,#mermaid-1722776679139 p{fill:#333;color:#333;}#mermaid-1722776679139 .node rect,#mermaid-1722776679139 .node circle,#mermaid-1722776679139 .node ellipse,#mermaid-1722776679139 .node polygon,#mermaid-1722776679139 .node path{fill:#ECECFF;stroke:#9370DB;stroke-width:1px;}#mermaid-1722776679139 .flowchart-label text{text-anchor:middle;}#mermaid-1722776679139 .node .katex path{fill:#000;stroke:#000;stroke-width:1px;}#mermaid-1722776679139 .node .label{text-align:center;}#mermaid-1722776679139 .node.clickable{cursor:pointer;}#mermaid-1722776679139 .arrowheadPath{fill:#333333;}#mermaid-1722776679139 .edgePath .path{stroke:#333333;stroke-width:2.0px;}#mermaid-1722776679139 .flowchart-link{stroke:#333333;fill:none;}#mermaid-1722776679139 .edgeLabel{background-color:#e8e8e8;text-align:center;}#mermaid-1722776679139 .edgeLabel rect{opacity:0.5;background-color:#e8e8e8;fill:#e8e8e8;}#mermaid-1722776679139 .labelBkg{background-color:rgba(232, 232, 232, 0.5);}#mermaid-1722776679139 .cluster rect{fill:#ffffde;stroke:#aaaa33;stroke-width:1px;}#mermaid-1722776679139 .cluster text{fill:#333;}#mermaid-1722776679139 .cluster span,#mermaid-1722776679139 p{color:#333;}#mermaid-1722776679139 div.mermaidTooltip{position:absolute;text-align:center;max-width:200px;padding:2px;font-family:"trebuchet ms",verdana,arial,sans-serif;font-size:12px;background:hsl(80, 100%, 96.2745098039%);border:1px solid #aaaa33;border-radius:2px;pointer-events:none;z-index:100;}#mermaid-1722776679139 .flowchartTitleText{text-anchor:middle;font-size:18px;fill:#333;}#mermaid-1722776679139 :root{--mermaid-font-family:"trebuchet ms",verdana,arial,sans-serif;}</style><g><marker orient="auto" markerHeight="12" markerWidth="12" markerUnits="userSpaceOnUse" refY="5" refX="6" viewBox="0 0 10 10" class="marker flowchart" id="mermaid-1722776679139_flowchart-pointEnd"><path style="stroke-width: 1; stroke-dasharray: 1, 0;" class="arrowMarkerPath" d="M 0 0 L 10 5 L 0 10 z"></path></marker><marker orient="auto" markerHeight="12" markerWidth="12" markerUnits="userSpaceOnUse" refY="5" refX="4.5" viewBox="0 0 10 10" class="marker flowchart" id="mermaid-1722776679139_flowchart-pointStart"><path style="stroke-width: 1; stroke-dasharray: 1, 0;" class="arrowMarkerPath" d="M 0 5 L 10 10 L 10 0 z"></path></marker><marker orient="auto" markerHeight="11" markerWidth="11" markerUnits="userSpaceOnUse" refY="5" refX="11" viewBox="0 0 10 10" class="marker flowchart" id="mermaid-1722776679139_flowchart-circleEnd"><circle style="stroke-width: 1; stroke-dasharray: 1, 0;" class="arrowMarkerPath" r="5" cy="5" cx="5"></circle></marker><marker orient="auto" markerHeight="11" markerWidth="11" markerUnits="userSpaceOnUse" refY="5" refX="-1" viewBox="0 0 10 10" class="marker flowchart" id="mermaid-1722776679139_flowchart-circleStart"><circle style="stroke-width: 1; stroke-dasharray: 1, 0;" class="arrowMarkerPath" r="5" cy="5" cx="5"></circle></marker><marker orient="auto" markerHeight="11" markerWidth="11" markerUnits="userSpaceOnUse" refY="5.2" refX="12" viewBox="0 0 11 11" class="marker cross flowchart" id="mermaid-1722776679139_flowchart-crossEnd"><path style="stroke-width: 2; stroke-dasharray: 1, 0;" class="arrowMarkerPath" d="M 1,1 l 9,9 M 10,1 l -9,9"></path></marker><marker orient="auto" markerHeight="11" markerWidth="11" markerUnits="userSpaceOnUse" refY="5.2" refX="-1" viewBox="0 0 11 11" class="marker cross flowchart" id="mermaid-1722776679139_flowchart-crossStart"><path style="stroke-width: 2; stroke-dasharray: 1, 0;" class="arrowMarkerPath" d="M 1,1 l 9,9 M 10,1 l -9,9"></path></marker><g class="root"><g class="clusters"><g id="Running" class="cluster default flowchart-label"><rect height="306.953125" width="354.84375" y="197.171875" x="0" ry="0" rx="0"></rect><g transform="translate(147.6171875, 197.171875)" class="cluster-label"><foreignobject height="22.390625" width="59.609375"><div style="display: inline-block; white-space: nowrap;" xmlns="http://www.w3.org/1999/xhtml"><span class="nodeLabel">Running</span></div></foreignobject></g></g></g><g class="edgePaths"><path marker-end="url(#mermaid-1722776679139_flowchart-pointEnd)" style="fill:none;" class="edge-thickness-normal edge-pattern-solid flowchart-link LS-0 LE-S" id="L-0-S-0" d="M176.344,15L176.344,21.033C176.344,27.065,176.344,39.13,176.344,50.312C176.344,61.494,176.344,71.792,176.344,76.941L176.344,82.091"></path><path marker-end="url(#mermaid-1722776679139_flowchart-pointEnd)" style="fill:none;" class="edge-thickness-normal edge-pattern-solid flowchart-link LS-C LE-R" id="L-C-R-0" d="M74.695,259.563L74.695,265.595C74.695,271.628,74.695,283.693,78.881,295.063C83.066,306.433,91.437,317.108,95.622,322.445L99.808,327.782"></path><path marker-end="url(#mermaid-1722776679139_flowchart-pointEnd)" style="fill:none;" class="edge-thickness-normal edge-pattern-solid flowchart-link LS-R LE-A" id="L-R-A-0" d="M117.738,369.344L117.738,375.376C117.738,381.409,117.738,393.474,126.825,405.156C135.912,416.838,154.086,428.137,163.173,433.787L172.26,439.436"></path><path marker-end="url(#mermaid-1722776679139_flowchart-pointEnd)" style="fill:none;" class="edge-thickness-normal edge-pattern-solid flowchart-link LS-A LE-C" id="L-A-C-0" d="M209.261,442.234L209.911,436.118C210.56,430.003,211.86,417.771,212.51,402.507C213.16,387.242,213.16,368.945,213.16,350.648C213.16,332.352,213.16,314.055,196.897,298.459C180.635,282.864,148.109,269.97,131.846,263.523L115.583,257.076"></path><path marker-end="url(#mermaid-1722776679139_flowchart-pointEnd)" style="fill:none;" class="edge-thickness-normal edge-pattern-solid flowchart-link LS-A LE-S" id="L-A-S-0" d="M232.858,442.234L241.122,436.118C249.386,430.003,265.915,417.771,274.179,402.507C282.443,387.242,282.443,368.945,282.443,350.648C282.443,332.352,282.443,314.055,282.443,295.758C282.443,277.461,282.443,259.164,282.443,242.733C282.443,226.302,282.443,211.737,282.443,198.422C282.443,185.107,282.443,173.042,271.567,161.382C260.692,149.723,238.94,138.47,228.064,132.843L217.188,127.217"></path><path marker-end="url(#mermaid-1722776679139_flowchart-pointEnd)" style="fill:none;" class="edge-thickness-normal edge-pattern-solid flowchart-link LS-S LE-C" id="L-S-C-0" d="M141.723,124.781L130.552,130.814C119.381,136.846,97.038,148.911,85.867,160.977C74.695,173.042,74.695,185.107,74.695,194.423C74.695,203.739,74.695,210.305,74.695,213.589L74.695,216.872"></path><path marker-end="url(#mermaid-1722776679139_flowchart-pointEnd)" style="fill:none;" class="edge-thickness-normal edge-pattern-solid flowchart-link LS-Running LE-S" id="L-Running-S-0" d="M160.781,197.172L160.781,191.139C160.781,185.107,160.781,173.042,162.251,161.826C163.72,150.611,166.659,140.246,168.128,135.063L169.598,129.88"></path></g><g class="edgeLabels"><g transform="translate(176.34375, 51.1953125)" class="edgeLabel"><g transform="translate(-44.9140625, -11.1953125)" class="label"><foreignobject height="22.390625" width="89.828125"><div style="display: inline-block; white-space: nowrap;" xmlns="http://www.w3.org/1999/xhtml"><span class="edgeLabel">new process</span></div></foreignobject></g></g><g transform="translate(74.6953125, 295.7578125)" class="edgeLabel"><g transform="translate(-54.6953125, -11.1953125)" class="label"><foreignobject height="22.390625" width="109.390625"><div style="display: inline-block; white-space: nowrap;" xmlns="http://www.w3.org/1999/xhtml"><span class="edgeLabel">process started</span></div></foreignobject></g></g><g transform="translate(117.73828125, 405.5390625)" class="edgeLabel"><g transform="translate(-45.359375, -11.1953125)" class="label"><foreignobject height="22.390625" width="90.71875"><div style="display: inline-block; white-space: nowrap;" xmlns="http://www.w3.org/1999/xhtml"><span class="edgeLabel">process died</span></div></foreignobject></g></g><g transform="translate(213.16015625, 350.6484375)" class="edgeLabel"><g transform="translate(-12.453125, -11.1953125)" class="label"><foreignobject height="22.390625" width="24.90625"><div style="display: inline-block; white-space: nowrap;" xmlns="http://www.w3.org/1999/xhtml"><span class="edgeLabel">yes</span></div></foreignobject></g></g><g transform="translate(282.443359375, 295.7578125)" class="edgeLabel"><g transform="translate(-8.8984375, -11.1953125)" class="label"><foreignobject height="22.390625" width="17.796875"><div style="display: inline-block; white-space: nowrap;" xmlns="http://www.w3.org/1999/xhtml"><span class="edgeLabel">no</span></div></foreignobject></g></g><g transform="translate(74.6953125, 160.9765625)" class="edgeLabel"><g transform="translate(-15.5625, -11.1953125)" class="label"><foreignobject height="22.390625" width="31.125"><div style="display: inline-block; white-space: nowrap;" xmlns="http://www.w3.org/1999/xhtml"><span class="edgeLabel">start</span></div></foreignobject></g></g><g transform="translate(160.97582, 160.2903)" class="edgeLabel"><g transform="translate(-15.125, -11.1953125)" class="label"><foreignobject height="22.390625" width="30.25"><div style="display: inline-block; white-space: nowrap;" xmlns="http://www.w3.org/1999/xhtml"><span class="edgeLabel">stop</span></div></foreignobject></g></g></g><g class="nodes"><g transform="translate(176.34375, 7.5)" data-id="0" data-node="true" id="flowchart-0-0" class="node default flowchart-label"><rect height="15" width="15" y="-7.5" x="-7.5" ry="5" rx="5" class="basic label-container"></rect><g transform="translate(0, 0)" class="label"><rect></rect><foreignobject height="0" width="0"><div style="display: inline-block; white-space: nowrap;" xmlns="http://www.w3.org/1999/xhtml"><span class="nodeLabel"></span></div></foreignobject></g></g><g transform="translate(117.73828125, 350.6484375)" data-id="R" data-node="true" id="flowchart-R-3" class="node default default flowchart-label"><rect height="37.390625" width="74.609375" y="-18.6953125" x="-37.3046875" ry="5" rx="5" class="basic label-container"></rect><g transform="translate(-29.8046875, -11.1953125)" class="label"><rect></rect><foreignobject height="22.390625" width="59.609375"><div style="display: inline-block; white-space: nowrap;" xmlns="http://www.w3.org/1999/xhtml"><span class="nodeLabel">Running</span></div></foreignobject></g></g><g transform="translate(74.6953125, 240.8671875)" data-id="C" data-node="true" id="flowchart-C-2" class="node default default flowchart-label"><rect height="37.390625" width="71.921875" y="-18.6953125" x="-35.9609375" ry="5" rx="5" class="basic label-container"></rect><g transform="translate(-28.4609375, -11.1953125)" class="label"><rect></rect><foreignobject height="22.390625" width="56.921875"><div style="display: inline-block; white-space: nowrap;" xmlns="http://www.w3.org/1999/xhtml"><span class="nodeLabel">Created</span></div></foreignobject></g></g><g transform="translate(206.48828125, 460.4296875)" data-id="A" data-node="true" id="flowchart-A-4" class="node default default flowchart-label"><polygon transform="translate(-113.35546875,18.6953125)" class="label-container" points="9.34765625,0 217.36328125,0 226.7109375,-18.6953125 217.36328125,-37.390625 9.34765625,-37.390625 0,-18.6953125"></polygon><g transform="translate(-96.5078125, -11.1953125)" class="label"><rect></rect><foreignobject height="22.390625" width="193.015625"><div style="display: inline-block; white-space: nowrap;" xmlns="http://www.w3.org/1999/xhtml"><span class="nodeLabel">autorestart/watch enabled?</span></div></foreignobject></g></g><g transform="translate(176.34375, 106.0859375)" data-id="S" data-node="true" id="flowchart-S-1" class="node default default flowchart-label"><rect height="37.390625" width="74.609375" y="-18.6953125" x="-37.3046875" ry="5" rx="5" class="basic label-container"></rect><g transform="translate(-29.8046875, -11.1953125)" class="label"><rect></rect><foreignobject height="22.390625" width="59.609375"><div style="display: inline-block; white-space: nowrap;" xmlns="http://www.w3.org/1999/xhtml"><span class="nodeLabel">Stopped</span></div></foreignobject></g></g></g></g></g></svg>
            </div>

            <h2 id="development" tabindex="-1"><a href="#development" data-id="development" class="anchor"><span>Development</span></a></h2>

            <h3 id="architecture" tabindex="-1"><a href="#architecture" data-id="architecture" class="anchor"><span>Architecture</span></a></h3>
            <p><code>pm</code> consists of two parts:</p>
            <ul>
              <li><b>cli client</b> - requests server, launches/stops shim processes</li>
              <li><b>shim</b> - monitors and restarts processes, handle watches, signals and shutdowns</li>
            </ul>

            <h3 id="pm-directory-structure" tabindex="-1"><a href="#pm-directory-structure" data-id="pm-directory-structure" class="anchor"><span>PM directory structure</span></a></h3>
            <p><code>pm</code> uses directory <code>$HOME/.pm</code> to store data by default. <code>PM_HOME</code> environment variable can be used to change this. Layout is following:</p>
            <pre data-lang="sh" class="language-sh"><code class="lang-sh language-sh" tabindex="0"><span class="token environment constant">$HOME</span>/.pm/
    ├──config.json <span class="token comment"># pm config file</span>
    ├──db/ <span class="token comment"># database tables</span>
    │   └──<span class="token operator">&lt;</span>ID<span class="token operator">&gt;</span> <span class="token comment"># process info</span>
    └──logs/ <span class="token comment"># processes logs</span>
      ├──<span class="token operator">&lt;</span>ID<span class="token operator">&gt;</span>.stdout <span class="token comment"># stdout of process with id ID</span>
      └──<span class="token operator">&lt;</span>ID<span class="token operator">&gt;</span>.stderr <span class="token comment"># stderr of process with id ID</span>
    </code></pre>

          <h3 id="differences-from-pm2" tabindex="-1"><a href="#differences-from-pm2" class="anchor"><span>Differences from </span></a> <a href="https://github.com/Unitech/pm2" target="_top">pm2</a></h3>
          <ul>
            <li><code>pm</code> is just a single binary, not dependent on <code>nodejs</code> and bunch of <code>js</code> scripts</li>
            <li><a href="https://jsonnet.org/" target="_top">jsonnet</a> configuration language, back compatible with <code>JSON</code>, and allows to thoroughly configure processes, e.g. separate environments without requiring corresponding mechanism in <code>pm</code> (others configuration languages might be added in future such as <code>Procfile</code>, <code>HCL</code>, etc.)</li>
            <li>supports only <code>linux</code> now</li>
            <li>I can fix problems/add features as I need, independent of whether they work or not in <code>pm2</code>, because I don’t know <code>js</code></li>
            <li>fast and convenient (I hope so)</li>
            <li>no specific integrations for <code>js</code></li>
          </ul>

          <h3 id="release" tabindex="-1"><a href="#release" class="anchor"><span>Release</span></a></h3>
          <p>On <code>master</code> branch:</p>
          <pre data-lang="sh" class="language-sh"><code class="lang-sh language-sh" tabindex="0"><span class="token function">git</span> tag v1.2.3
    <span class="token function">git</span> push <span class="token parameter variable">--tags</span>
    goreleaser release <span class="token parameter variable">--clean</span></code></pre>

      </section>
    </main></body>
|||,
  "readme.md": importstr "../readme.md",
}
