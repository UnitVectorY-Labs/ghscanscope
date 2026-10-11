package web

// The inline symbol paths are from Tabler Icons, licensed under the MIT License.
const pageTemplate = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{{.Title}} · ghscanscope</title>
<script src="https://unpkg.com/htmx.org@4.0.0/dist/htmx.min.js" integrity="sha384-BvJpBiO8Kh31EqtJe5DRIeWrHWnCGkwytKs9NKFi86Hhw96dEqdEMzZDeK9iEGTc" crossorigin="anonymous"></script>
<script>
(function () {
  function savedTheme() { try { return localStorage.getItem('ghscanscope-theme'); } catch (_) { return null; } }
  var systemTheme = window.matchMedia('(prefers-color-scheme: dark)');
  var saved = savedTheme();
  document.documentElement.dataset.theme = saved === 'dark' || saved === 'light' ? saved : (systemTheme.matches ? 'dark' : 'light');
  function updateThemeButton() {
    var button = document.getElementById('theme-toggle');
    if (!button) return;
    var dark = document.documentElement.dataset.theme === 'dark';
    button.setAttribute('aria-pressed', String(dark));
    button.setAttribute('aria-label', dark ? 'Use light mode' : 'Use dark mode');
    button.title = dark ? 'Use light mode' : 'Use dark mode';
  }
  window.toggleTheme = function () {
    var theme = document.documentElement.dataset.theme === 'dark' ? 'light' : 'dark';
    document.documentElement.dataset.theme = theme;
    try { localStorage.setItem('ghscanscope-theme', theme); } catch (_) {}
    updateThemeButton();
  };
  systemTheme.addEventListener('change', function (event) {
    if (!savedTheme()) { document.documentElement.dataset.theme = event.matches ? 'dark' : 'light'; updateThemeButton(); }
  });
  document.addEventListener('DOMContentLoaded', updateThemeButton);
  document.addEventListener('htmx:after:swap', updateThemeButton);
  if (document.readyState !== 'loading') queueMicrotask(updateThemeButton);
})();
</script>
<style>

:root{color-scheme:light;--blue:#2563eb;--blue-dark:#1e40af;--blue-soft:#eff6ff;--blue-border:#c6d7f2;--orange:#c45b17;--orange-soft:#fff4e9;--ink:#1e2530;--muted:#69717e;--line:#e4e7eb;--surface:#fff;--canvas:#fff;--soft:#f7f8fa;--button-ink:#fff;--critical:#9a3412;--high:#b54708;--medium:#a85b16;--low:#2563eb;--radius:10px;--scanner-bg:#f0f1f3;--scanner-ink:#555c68;--scanner-mark:#888e98;--codeql-bg:#edf0f4;--codeql-ink:#4e5c6d;--codeql-mark:#7e8c9e;--semgrep-bg:#f0efec;--semgrep-ink:#605e56;--semgrep-mark:#938e7e;--govulncheck-bg:#f0eef2;--govulncheck-ink:#625b6b;--govulncheck-mark:#91869a;--zizmor-bg:#f3eeea;--zizmor-ink:#6c5b50;--zizmor-mark:#a28d7d;font-family:Inter,ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;color:var(--ink);background:var(--canvas);font-synthesis:none}
:root[data-theme="dark"]{color-scheme:dark;--blue:#8ab6ff;--blue-dark:#b4cfff;--blue-soft:#1b2a42;--blue-border:#385173;--orange:#e89a5e;--orange-soft:#36271e;--ink:#e4e7ec;--muted:#a2aab7;--line:#323842;--surface:#1a1e25;--canvas:#13161c;--soft:#21262f;--button-ink:#131c2b;--critical:#ffc394;--high:#f1af7c;--medium:#dda168;--low:#8ab6ff;--scanner-bg:#292d35;--scanner-ink:#c5cad4;--scanner-mark:#89909d;--codeql-bg:#27303c;--codeql-ink:#c1cbd9;--codeql-mark:#90a0b5;--semgrep-bg:#302f2a;--semgrep-ink:#d2d0c4;--semgrep-mark:#a7a18b;--govulncheck-bg:#302c36;--govulncheck-ink:#d0c8da;--govulncheck-mark:#a497b2;--zizmor-bg:#342d28;--zizmor-ink:#d7cabe;--zizmor-mark:#b8a18d}
*{box-sizing:border-box}
body{margin:0}
a{color:var(--blue);text-decoration:none}
a:hover{text-decoration:underline}
.icon{width:1.15rem;height:1.15rem;stroke-width:1.8;fill:none;stroke:currentColor;stroke-linecap:round;stroke-linejoin:round;vertical-align:-.2rem}
.icon-filled{fill:currentColor;stroke:none}
.shell{max-width:1440px;margin:0 auto;padding:0 28px}
.topbar{background:var(--surface);color:var(--ink);border-bottom:1px solid var(--line)}
.topbar-row{height:70px;display:flex;align-items:center;gap:28px}
.brand{display:flex;align-items:center;gap:10px;font-weight:750;font-size:1.08rem;letter-spacing:-.02em;color:var(--ink)}
.brand-mark{display:grid;place-items:center;width:34px;height:34px;border-radius:9px;background:var(--blue);color:var(--button-ink)}
.tabs{display:flex;align-self:stretch;gap:4px}
.tab{display:flex;align-items:center;gap:8px;color:var(--muted);padding:0 16px;border-bottom:3px solid transparent;margin-bottom:-3px;font-weight:600}
.tab:hover{color:var(--blue);text-decoration:none;background:var(--blue-soft)}
.tab.active{color:var(--blue);border-color:var(--orange)}
.sync-org{margin-left:auto;display:flex;gap:8px}
.sync-org select{width:210px;background:var(--surface);color:var(--ink);border:1px solid var(--line);border-radius:7px;padding:8px 10px}
.btn{display:inline-flex;align-items:center;justify-content:center;gap:7px;border:0;border-radius:7px;padding:9px 13px;font-weight:650;cursor:pointer;background:var(--blue);color:var(--button-ink)}
.btn:hover{background:var(--blue-dark);text-decoration:none}
.btn.secondary{background:var(--surface);color:var(--blue);border:1px solid var(--blue-border)}
.btn.icon-only{padding:9px}
.breadcrumbs{font-size:.75rem;color:var(--muted);margin-bottom:8px}
.breadcrumbs a{color:var(--muted)}
.breadcrumbs span{padding:0 6px;color:var(--muted)}
.page-head{display:flex;justify-content:space-between;gap:24px;align-items:end;padding:30px 0 20px}
.page-head>div:first-child,.detail-grid>*{min-width:0}
.page-head h1{font-size:1.85rem;line-height:1.15;margin:0;letter-spacing:-.035em;overflow-wrap:anywhere}
.eyebrow{text-transform:uppercase;letter-spacing:.1em;font-size:.72rem;color:var(--blue);font-weight:750;margin-bottom:5px}
.subtitle{color:var(--muted);margin:7px 0 0}
.secondary-text{display:block;color:var(--muted);font-size:.73rem;margin-top:3px}
.notice{display:flex;gap:9px;align-items:center;background:var(--blue-soft);color:var(--blue-dark);border:1px solid var(--blue-border);padding:11px 14px;border-radius:9px;margin-bottom:16px}
.panel{background:var(--surface);border:1px solid var(--line);border-radius:var(--radius);box-shadow:0 1px 2px #10182808;margin-bottom:24px}
.panel-head{display:flex;align-items:center;justify-content:space-between;gap:16px;padding:16px 18px;border-bottom:1px solid var(--line)}
.panel-head h2{font-size:1rem;margin:0}
.count{display:inline-block;border-radius:999px;background:var(--blue-soft);color:var(--blue-dark);padding:3px 8px;font-size:.74rem;font-weight:700}
.table-wrap{overflow:auto;min-height:180px}
table{width:100%;border-collapse:collapse;font-size:.875rem}
th{font-size:.69rem;text-transform:uppercase;letter-spacing:.055em;color:var(--muted);font-weight:700;background:var(--soft);white-space:nowrap}
th,td{text-align:left;padding:12px 14px;border-bottom:1px solid var(--line);vertical-align:middle}
tbody tr:last-child td{border-bottom:0}
tbody tr:hover{background:var(--soft)}
.primary{font-weight:650;color:var(--ink)}
.mono{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:.79rem}
.severity{display:inline-flex;align-items:center;gap:5px;border-radius:999px;padding:4px 8px;font-size:.72rem;font-weight:750;text-transform:capitalize}
.severity:before{content:"";width:6px;height:6px;border-radius:50%;background:currentColor}
.severity-critical{background:var(--orange-soft);color:var(--critical)}
.severity-high{background:var(--orange-soft);color:var(--high)}
.severity-medium{background:var(--orange-soft);color:var(--medium)}
.severity-low{background:var(--blue-soft);color:var(--low)}
.severity-other{background:var(--soft);color:var(--muted)}
.count-total{color:var(--muted);font-weight:450}
.zero{color:var(--muted)}
.empty{padding:45px 20px;text-align:center;color:var(--muted)}
.detail-grid{display:grid;grid-template-columns:minmax(0,2fr) minmax(290px,1fr);gap:20px}
.detail-section{padding:20px}
.detail-section h2{font-size:.95rem;margin:0 0 14px}
.message{font-size:1rem;line-height:1.6;background:var(--orange-soft);border-left:3px solid var(--orange);padding:14px 16px;border-radius:0 8px 8px 0}
.description{line-height:1.65;color:var(--ink)}
.location{display:block;background:#111827;color:#e5e7eb;border-radius:9px;padding:14px 16px;overflow:auto}
.location:hover{text-decoration:none}
.location strong{color:#fdba74}
.metadata{margin:0}
.metadata>div{display:grid;grid-template-columns:120px 1fr;gap:12px;padding:10px 0;border-bottom:1px solid var(--line)}
.metadata>div:last-child{border-bottom:0}
.metadata dt{color:var(--muted);font-size:.75rem}
.metadata dd{margin:0;font-size:.84rem;overflow-wrap:anywhere}
.tags{display:flex;flex-wrap:wrap;gap:6px}
.tag{background:var(--soft);border-radius:5px;padding:4px 7px;font-size:.72rem;color:var(--muted)}
footer{padding:8px 0 30px;color:var(--muted);font-size:.72rem;text-align:center}
footer p{margin:3px 0}
@media(max-width:1050px){
.detail-grid{grid-template-columns:1fr}
}
@media(max-width:720px){
.shell{padding:0 14px}
.topbar-row{height:auto;min-height:64px;flex-wrap:wrap;padding:12px 0}
.tabs{order:3;width:100%;height:48px}
.tab{flex:1;justify-content:center}
.sync-org{margin-left:auto}
.sync-org select{width:150px}
.page-head{padding-top:22px;flex-direction:column;align-items:flex-start;gap:10px}
th,td{white-space:nowrap}
.detail-section td{white-space:normal}
}
.htmx-request{opacity:.65;transition:opacity .15s}
@media(max-width:720px){
.topbar-row{column-gap:8px}
.brand{gap:8px}
.brand-mark{margin-left:2px}
.sync-org{gap:5px}
.sync-org select{width:137px}
}
.severity-unknown{background:var(--soft);color:var(--muted)}
</style><style>
.page-head{align-items:start;padding:30px 0 24px}
.page-head h1{font-size:1.8rem;max-width:980px}
.eyebrow{color:var(--muted);font-size:.68rem}
.subtitle{font-size:.9rem;max-width:890px;line-height:1.6}
.sync-time{font-size:.76rem;color:var(--muted);white-space:nowrap;padding-top:10px}
.scope-summary{display:flex;align-items:center;gap:30px;padding:18px 22px;background:var(--surface);border:1px solid var(--line);border-radius:10px;margin-bottom:16px}
.scope-summary>div:not(.priority-breakdown){display:flex;flex-direction:column;gap:3px;min-width:135px}
.scope-summary strong{font-size:1.6rem;letter-spacing:-.03em}
.scope-summary span:not(.severity):not(.count-total){font-size:.76rem;color:var(--muted)}
.priority-breakdown{display:flex;flex-wrap:wrap;gap:10px;margin-left:auto}
.filter-bar{padding:14px 16px;position:relative;z-index:10;margin-bottom:22px}
.filter-top{display:flex;gap:9px;align-items:center;flex-wrap:wrap}
.search{flex:1;min-width:240px}
.search input,.facet-menu>input{width:100%;font:inherit;border:1px solid var(--line);border-radius:6px;padding:10px 12px;background:var(--soft);color:var(--ink);font-size:.84rem}
.search input:focus{outline:2px solid var(--blue-border);outline-offset:1px}
.search-submit{padding:9px 12px}
.facet{position:relative}
.facet summary{list-style:none;cursor:pointer;padding:9px 12px;border:1px solid var(--line);border-radius:6px;font-size:.8rem;font-weight:650;display:flex;gap:14px;align-items:center}
.facet summary::-webkit-details-marker{display:none}
.facet.active summary,.facet[open] summary{background:var(--blue-soft);border-color:var(--blue-border);color:var(--blue-dark)}
.facet-menu{position:absolute;top:44px;right:0;width:240px;max-height:340px;overflow:auto;background:var(--surface);border:1px solid var(--line);border-radius:8px;padding:10px;box-shadow:0 12px 30px #17203326}
.facet-menu.wide{width:340px}
.facet-menu label{display:flex;align-items:start;gap:8px;font-size:.8rem;padding:8px 5px;overflow-wrap:anywhere;cursor:pointer}
.facet-menu label[hidden]{display:none}
.facet-menu input[type=checkbox]{accent-color:var(--blue);margin-top:3px;flex-shrink:0}
.facet-menu h3{font-size:.73rem;text-transform:uppercase;color:var(--muted);padding:8px 5px;border-top:1px solid var(--line)}
.clear-filters{font-size:.78rem;padding:8px}
.filter-chips{display:flex;gap:7px;align-items:center;flex-wrap:wrap;border-top:1px solid var(--line);margin-top:12px;padding-top:12px;font-size:.73rem}
.filter-chips>span{color:var(--muted);margin-right:4px}
.filter-chips a{display:inline-flex;gap:8px;align-items:center;background:var(--soft);border:1px solid var(--line);color:var(--ink);padding:5px 8px;border-radius:5px;overflow-wrap:anywhere}
.filter-chips a:hover{background:var(--blue-soft);text-decoration:none}
.panel-head p{font-size:.77rem;color:var(--muted);margin:5px 0 0;line-height:1.5}
.panel-head{padding:18px 20px}
.issue-table{table-layout:fixed;min-width:850px}
.issue-table th:nth-child(1){width:45%}
.issue-table th:nth-child(2){width:16%}
.issue-table th:nth-child(3){width:13%}
.issue-table th:nth-child(4){width:12%}
.issue-table th:nth-child(5){width:14%}
th a{color:var(--muted)}
td{padding:18px 20px}
th{padding:12px 20px}
.issue-title{display:block;font-size:.9rem;line-height:1.45;color:var(--ink)}
.issue-title:hover{color:var(--blue)}
.issue-cell .mono{font-size:.68rem;overflow-wrap:anywhere}
.scanner{display:inline-flex;align-items:center;vertical-align:middle;background:var(--scanner-bg);color:var(--scanner-ink);border:1px solid var(--line);border-left:3px solid var(--scanner-mark);border-radius:5px;padding:4px 8px;font-size:.76rem;font-weight:750;line-height:1.25;margin-right:5px}
.scanner-codeql{--scanner-bg:var(--codeql-bg);--scanner-ink:var(--codeql-ink);--scanner-mark:var(--codeql-mark)}
.scanner-semgrep{--scanner-bg:var(--semgrep-bg);--scanner-ink:var(--semgrep-ink);--scanner-mark:var(--semgrep-mark)}
.scanner-govulncheck{--scanner-bg:var(--govulncheck-bg);--scanner-ink:var(--govulncheck-ink);--scanner-mark:var(--govulncheck-mark)}
.scanner-zizmor{--scanner-bg:var(--zizmor-bg);--scanner-ink:var(--zizmor-ink);--scanner-mark:var(--zizmor-mark)}
.scanner-attribution{display:flex;align-items:center;flex-wrap:wrap;gap:9px;margin:12px 0 14px}
.scanner-attribution>span:first-child{font-size:.73rem;color:var(--muted);font-weight:600}
.scanner-attribution .scanner{padding:7px 11px;font-size:.94rem;border-left-width:4px}
.finding-scanner{display:flex;align-items:center;flex-wrap:wrap;gap:7px;margin-top:9px}
.finding-scanner .mono{margin-top:0}
.theme-toggle{display:grid;place-items:center;flex-shrink:0;width:36px;height:36px;padding:0;border:1px solid var(--line);border-radius:7px;background:var(--surface);color:var(--muted);cursor:pointer;margin-left:0}
.theme-toggle:hover{background:var(--blue-soft);color:var(--blue)}
.theme-toggle .sun{display:none}
[data-theme="dark"] .theme-toggle .sun{display:block}
[data-theme="dark"] .theme-toggle .moon{display:none}
@media(max-width:720px){.topbar-row{gap:8px}.brand{font-size:.95rem}.brand-mark{width:30px;height:30px}.sync-org select{width:120px}.sync-org{margin-left:auto}.topbar .icon-only{padding:8px}.theme-toggle{width:32px;height:32px}}
.row-description{color:var(--muted);line-height:1.5;font-size:.76rem;margin:7px 0 0;display:-webkit-box;-webkit-line-clamp:2;-webkit-box-orient:vertical;overflow:hidden}
.metric{font-size:1.25rem;font-weight:750;color:var(--ink);font-variant-numeric:tabular-nums}
.date-cell{font-size:.76rem;color:var(--muted)}
.issue-table .tags,.estate-table .tags{gap:5px}
.issue-table .tags{flex-direction:column;align-items:start}
.estate-table{table-layout:fixed;min-width:700px}
.estate-table th:nth-child(1){width:42%}
.estate-table th:nth-child(2){width:30%}
.estate-table th:nth-child(3){width:15%}
.estate-table th:nth-child(4){width:13%}
.estate-table td:last-child{font-size:.78rem}
.repo-context{display:flex;align-items:center;gap:22px;margin:0 0 18px;font-size:.8rem;color:var(--muted)}
.repo-context form{margin-left:auto}
.issue-overview{display:grid;grid-template-columns:1fr 260px;gap:30px;padding:25px}
.overview-copy .description{margin:10px 0 16px;font-size:1rem;white-space:pre-line}
.overview-copy>.secondary-text{margin-top:20px;line-height:1.6}
.overview-impact{border-left:1px solid var(--line);padding-left:25px;display:flex;flex-direction:column;justify-content:center}
.overview-impact>strong{font-size:2.5rem;letter-spacing:-.05em}
.overview-impact>span{font-size:.8rem;color:var(--muted)}
.example-message{border-left:3px solid var(--blue-border);padding:2px 12px;background:var(--soft);margin:14px 0}
.example-message p{font-size:.86rem;line-height:1.6;margin:6px 0;white-space:pre-wrap;overflow-wrap:anywhere}
.overview-impact>.tags{gap:6px}
.overview-impact>a{margin-top:14px;color:var(--blue)}
.overview-impact p{font-size:.85rem;line-height:1.5}
.repository-impact{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));max-height:360px;overflow:auto}
.repository-impact>a{display:flex;gap:14px;justify-content:space-between;align-items:center;padding:16px 20px;border-bottom:1px solid var(--line);color:var(--ink);font-size:.82rem}
.repository-impact>a:hover{background:var(--blue-soft);text-decoration:none}
.repository-impact>a>span:first-child{flex:1;min-width:0;overflow-wrap:anywhere}
.repository-impact>a>span:nth-child(2){font-size:.75rem;text-align:right}
.repository-impact .tags{margin-top:5px;justify-content:end}
.repository-impact .severity{font-size:.65rem}
.finding-table{table-layout:fixed;min-width:760px}
.finding-table th:nth-last-child(2){width:120px}
.finding-table th:last-child{width:145px}
.finding-table th:first-child:not(:nth-last-child(3)){width:26%}
.evidence-cell .issue-title{white-space:normal;overflow-wrap:anywhere}
.evidence-cell .mono{margin-top:8px;overflow-wrap:anywhere}
.evidence-cell>a.secondary-text{margin-top:8px;color:var(--blue)}
.occurrence-heading{display:flex;align-items:center;justify-content:space-between;margin-bottom:22px}
.occurrence-heading .secondary-text{margin-top:8px}
.detail-section h2{font-size:1rem}
.evidence .message{background:var(--soft);border-color:var(--blue-border);color:var(--ink);white-space:pre-wrap;overflow-wrap:anywhere}
.description{white-space:pre-line;overflow-wrap:anywhere}
.location{overflow-wrap:anywhere;white-space:normal;line-height:1.6;background:#18283f}
.location strong{color:#e0edff}
.location .secondary-text{color:#afc1dc}
.related-link{display:block;border-top:1px solid var(--line);margin-top:22px;padding-top:15px;font-size:.82rem;font-weight:650}
.scan-context summary{cursor:pointer;font-size:.8rem;font-weight:650}
.scan-context .metadata{margin-top:12px}
.sr-only{position:absolute;width:1px;height:1px;padding:0;margin:-1px;overflow:hidden;clip:rect(0,0,0,0);white-space:nowrap;border:0}
.breadcrumbs{margin-bottom:14px}
.breadcrumbs a{color:var(--blue)}
.severity{white-space:nowrap}
a:focus-visible,summary:focus-visible,button:focus-visible{outline:2px solid var(--blue);outline-offset:3px}
.topbar .sync-org select{font-size:.8rem}
@media(max-width:1000px){
.sync-time{white-space:normal}
.scope-summary{flex-wrap:wrap}
.priority-breakdown{margin-left:0}
.issue-overview{grid-template-columns:1fr 220px}
.page-head h1{font-size:1.5rem}
}
@media(max-width:720px){
.scope-summary{gap:18px}
.filter-top{gap:7px}
.search{min-width:200px}
.facet-menu,.facet-menu.wide{width:min(320px,85vw);left:0;right:auto}
.facet:last-of-type .facet-menu{left:auto;right:0}
.repository-impact{grid-template-columns:1fr}
.issue-overview{grid-template-columns:1fr}
.overview-impact{border-left:0;border-top:1px solid var(--line);padding:16px 0 0}
.overview-impact>strong{font-size:2rem}
.scope-summary>div:not(.priority-breakdown){min-width:100px}
.occurrence-heading{gap:12px;align-items:start}
.repo-context{flex-wrap:wrap;gap:12px}
.repo-context form{margin-left:0}
.issue-table td,.estate-table td,.finding-table td{white-space:normal}
.page-head{gap:10px}
.sync-time{padding-top:0}
.panel-head{flex-wrap:wrap}
}
</style>
</head><body>
<svg xmlns="http://www.w3.org/2000/svg" style="display:none"><symbol id="i-chevron" viewBox="0 0 24 24"><path d="M9 6l6 6l-6 6"/></symbol><symbol id="i-shield" viewBox="0 0 24 24"><path d="M12 3l8 4v5c0 5-3.5 8.5-8 10-4.5-1.5-8-5-8-10V7l8-4z"/><path d="M9 12l2 2 4-4"/></symbol><symbol id="i-alert" viewBox="0 0 24 24"><path d="M12 9v4m0 4v.01"/><path d="M5.1 19h13.8a2 2 0 0 0 1.7-3L13.7 4a2 2 0 0 0-3.4 0L3.4 16a2 2 0 0 0 1.7 3z"/></symbol><symbol id="i-repo" viewBox="0 0 24 24"><path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/></symbol><symbol id="i-refresh" viewBox="0 0 24 24"><path d="M20 11a8.1 8.1 0 0 0-15.5-2M4 4v5h5"/><path d="M4 13a8.1 8.1 0 0 0 15.5 2M20 20v-5h-5"/></symbol><symbol id="i-clock" viewBox="0 0 24 24"><circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/></symbol><symbol id="i-external" viewBox="0 0 24 24"><path d="M12 6H6a2 2 0 0 0-2 2v10a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2v-6"/><path d="M11 13l9-9m-5 0h5v5"/></symbol><symbol id="i-code" viewBox="0 0 24 24"><path d="M7 8l-4 4 4 4m10-8 4 4-4 4M14 4l-4 16"/></symbol><symbol id="i-check" viewBox="0 0 24 24"><path d="M5 12l5 5L20 7"/></symbol><symbol id="i-filter" viewBox="0 0 24 24"><path d="M4 4h16v2l-6 7v5l-4 2v-7L4 6z"/></symbol><symbol id="i-github" viewBox="0 0 24 24"><path d="M5.315 2.1c.791 -.113 1.9 .145 3.333 .966l.272 .161l.16 .1l.397 -.083a13.3 13.3 0 0 1 4.59 -.08l.456 .08l.396 .083l.161 -.1c1.385 -.84 2.487 -1.17 3.322 -1.148l.164 .008l.147 .017l.076 .014l.05 .011l.144 .047a1 1 0 0 1 .53 .514a5.2 5.2 0 0 1 .397 2.91l-.047 .267l-.046 .196l.123 .163c.574 .795 .93 1.728 1.03 2.707l.023 .295l.007 .272c0 3.855 -1.659 5.883 -4.644 6.68l-.245 .061l-.132 .029l.014 .161l.008 .157l.004 .365l-.002 .213l-.003 3.834a1 1 0 0 1 -.883 .993l-.117 .007h-6a1 1 0 0 1 -.993 -.883l-.007 -.117v-.734c-1.818 .26 -3.03 -.424 -4.11 -1.878l-.535 -.766c-.28 -.396 -.455 -.579 -.589 -.644l-.048 -.019a1 1 0 0 1 .564 -1.918c.642 .188 1.074 .568 1.57 1.239l.538 .769c.76 1.079 1.36 1.459 2.609 1.191l.001 -.678l-.018 -.168a5.03 5.03 0 0 1 -.021 -.824l.017 -.185l.019 -.12l-.108 -.024c-2.976 -.71 -4.703 -2.573 -4.875 -6.139l-.01 -.31l-.004 -.292a5.6 5.6 0 0 1 .908 -3.051l.152 -.222l.122 -.163l-.045 -.196a5.2 5.2 0 0 1 .145 -2.642l.1 -.282l.106 -.253a1 1 0 0 1 .529 -.514l.144 -.047l.154 -.03z"/></symbol><symbol id="i-moon" viewBox="0 0 24 24"><path d="M20.9 13A9 9 0 0 1 11 3.1 9 9 0 1 0 20.9 13z"/></symbol><symbol id="i-sun" viewBox="0 0 24 24"><circle cx="12" cy="12" r="4"/><path d="M12 2v2m0 16v2M2 12h2m16 0h2M5 5l1.5 1.5m11 11L19 19M5 19l1.5-1.5m11-11L19 5"/></symbol></svg>
<header class="topbar"><div class="shell topbar-row"><a class="brand" href="/alerts"><span class="brand-mark"><svg class="icon"><use href="#i-shield"/></svg></span>ghscanscope</a><nav class="tabs" aria-label="Primary"><a class="tab {{if or (eq .View "alerts") (eq .View "alert") (eq .View "rule")}}active{{end}}" href="/alerts"><svg class="icon"><use href="#i-alert"/></svg>Alerts</a><a class="tab {{if or (eq .View "repositories") (eq .View "repository")}}active{{end}}" href="/repositories"><svg class="icon"><use href="#i-repo"/></svg>Repositories</a></nav>{{if .Organizations}}<form class="sync-org" method="post" action="/sync"><select name="org" required aria-label="Organization to sync">{{if gt (len .Organizations) 1}}<option value="" selected disabled>Sync organization…</option>{{end}}{{range .Organizations}}<option value="{{.Login}}" {{if eq (len $.Organizations) 1}}selected{{end}}>{{.Login}}</option>{{end}}</select><button class="btn icon-only" title="Sync selected organization" aria-label="Sync selected organization"><svg class="icon"><use href="#i-refresh"/></svg></button></form>
{{end}}<button id="theme-toggle" class="theme-toggle" type="button" onclick="toggleTheme()" aria-label="Use dark mode" aria-pressed="false" title="Use dark mode"><svg class="icon moon" aria-hidden="true"><use href="#i-moon"/></svg><svg class="icon sun" aria-hidden="true"><use href="#i-sun"/></svg></button></div></header>
<main id="main-content" class="shell">
<div class="page-head"><div>{{if or (eq .View "rule") (eq .View "repository") (eq .View "alert")}}<div class="breadcrumbs"><a href="{{.BackURL}}">← Back to {{.BackLabel}}</a>{{if eq .View "alert"}}<span>/</span>{{.Alert.Repository}}<span>/</span>GitHub alert #{{.Alert.GitHubID}}{{end}}</div>{{end}}<div class="eyebrow">{{.Eyebrow}}</div>{{if eq .View "rule"}}<div class="scanner-attribution"><span>Detected by</span>{{template "scanner-badge" .RuleTool}}</div>{{else if eq .View "alert"}}<div class="scanner-attribution"><span>Detected by</span>{{template "scanner-badge" .Alert.Tool}}</div>{{end}}<h1>{{.Title}}</h1><p class="subtitle">{{if eq .View "alerts"}}Find the issues worth fixing across your repositories. Each row is one scanner rule, with its affected repositories and open occurrences.{{else if eq .View "repositories"}}Compare open findings by repository, then inspect the issues behind the counts.{{else if eq .View "repository"}}{{.Repository.Description}}{{else if eq .View "rule"}}<span class="mono">{{.RuleID}}</span>{{else if eq .View "alert"}}<span class="mono">{{.Alert.RuleID}}</span>{{end}}</p></div>{{if .Stats.LastSync}}<div class="sync-time"><svg class="icon"><use href="#i-clock"/></svg> Synced {{.Stats.LastSync.Format "Jan 2, 3:04 PM MST"}}<span class="secondary-text">Locally stored open code-scanning alerts</span></div>{{end}}</div>
{{if eq .View "rule"}}<section class="panel issue-overview"><div class="overview-copy"><div class="eyebrow">What this scanner reports</div>{{if .RuleDescription}}<p class="description">{{.RuleDescription}}</p>{{else}}<p class="description">The scanner did not provide a rule description. Read the individual finding messages below for the reported evidence.</p>{{end}}{{if .RuleMessage}}<div class="example-message"><span class="secondary-text">Example finding in this scope</span><p>{{.RuleMessage}}</p></div>{{end}}<div class="tags">{{range .RuleTags}}<span class="tag">{{.}}</span>{{end}}</div><p class="secondary-text">Grouped by {{.RuleTool}} and rule {{.RuleID}}. Occurrences share a rule; each location still needs individual review.</p></div><div class="overview-impact"><strong>{{countOf .Stats.Open .Stats.Totals.Open}}</strong><span>open occurrences matching / total before filters</span><p>across <b>{{countOf .RuleRepositoryCount .Stats.Totals.AffectedRepositories}}</b> affected repositories</p>{{template "repo-priorities" .Stats}}<a class="secondary-text" href="{{.ResetURL}}">View all occurrences of this rule →</a></div></section>
{{end}}
{{if .Notice}}<div class="notice">{{.Notice}}</div>{{end}}
{{if or (eq .View "alerts") (eq .View "repositories") (eq .View "repository") (eq .View "rule")}}
{{if ne .View "rule"}}<section class="scope-summary" aria-label="Summary statistics"><div><strong>{{countOf .Stats.Open .Stats.Totals.Open}}</strong><span>open occurrences · matching / total before filters</span></div><div><strong>{{if eq .View "repositories"}}{{countOf (len .Repositories) .Stats.Totals.TotalRepositories}}{{else}}{{countOf .Stats.AffectedRepositories .Stats.Totals.AffectedRepositories}}{{end}}</strong><span>{{if eq .View "repositories"}}repositories{{else}}affected repositories{{end}} · matching / total before filters</span></div><div class="priority-breakdown">{{template "stats-priorities" .Stats}}</div></section>
{{end}}
<form class="filter-bar panel" method="get" action="{{.Action}}" hx-get="{{.Action}}" hx-target="#main-content" hx-select="#main-content" hx-swap="outerHTML" hx-push-url="true" aria-label="Filter findings">
<input type="hidden" name="scope" value="all"><input type="hidden" name="sort" value="{{.SortColumn}}"><input type="hidden" name="dir" value="{{.SortDirection}}"><input type="hidden" name="filter_open" value="{{.FilterOpen}}"><input type="hidden" name="from" value="{{.BackURL}}">{{if eq .View "rule"}}<input type="hidden" name="tool" value="{{.RuleTool}}"><input type="hidden" name="rule" value="{{.RuleID}}"><input type="hidden" name="alert_sort" value="{{.SortColumn}}"><input type="hidden" name="alert_dir" value="{{.SortDirection}}">{{end}}
<div class="filter-top"><label class="search"><span class="sr-only">Search findings</span><input id="finding-search" hx-preserve="true" name="q" type="search" value="{{.Query}}" placeholder="{{if eq .View "repositories"}}Search repositories…{{else}}Search findings, rules, paths…{{end}}" hx-trigger="input changed delay:350ms, search" hx-get="{{.Action}}" hx-include="closest form" hx-target="#main-content" hx-select="#main-content" hx-swap="outerHTML" hx-push-url="true"></label><button class="btn secondary search-submit">Search</button>
<details class="facet {{if .Filters.priority}}active{{end}}" {{if eq .FilterOpen "priority"}}open{{end}}><summary aria-label="Filter priorities">Priority <span>⌄</span></summary><div class="facet-menu">{{range .Severities}}<label><input type="checkbox" id="priority-{{.}}" name="priority" value="{{.}}" {{if selected $.Filters.priority .}}checked{{end}} onchange="this.form.elements.filter_open.value='priority';this.form.requestSubmit()">{{.}}</label>{{end}}</div></details>

{{if ne .View "rule"}}<details class="facet {{if .Filters.tool}}active{{end}}" {{if eq .FilterOpen "tool"}}open{{end}}><summary aria-label="Filter tools">Scanner <span>⌄</span></summary><div class="facet-menu">{{range .Tools}}<label><input type="checkbox" id="tool-{{urlquery .}}" name="tool" value="{{.}}" {{if selected $.Filters.tool .}}checked{{end}} onchange="this.form.elements.filter_open.value='tool';this.form.requestSubmit()">{{.}}</label>{{end}}</div></details>
{{end}}
{{if ne .View "repository"}}<details class="facet {{if .Filters.repo}}active{{end}}" {{if eq .FilterOpen "repo"}}open{{end}}><summary aria-label="Filter repositories">Repository <span>⌄</span></summary><div class="facet-menu wide"><input type="search" placeholder="Find a repository…" aria-label="Find a repository in filters" oninput="for (const label of this.parentElement.querySelectorAll('label')) label.hidden=!label.textContent.toLowerCase().includes(this.value.toLowerCase())">{{range .AllRepositories}}<label><input type="checkbox" id="repo-{{urlquery .FullName}}" name="repo" value="{{.FullName}}" {{if selected $.Filters.repo .FullName}}checked{{end}} onchange="this.form.elements.filter_open.value='repo';this.form.requestSubmit()">{{.FullName}}</label>{{end}}</div></details>
{{else}}{{range .Filters.repo}}<input type="hidden" name="repo" value="{{.}}">{{end}}{{end}}
<details class="facet" {{if eq .FilterOpen "more"}}open{{end}}><summary>More filters <span>⌄</span></summary><div class="facet-menu wide">{{if ne .View "rule"}}<h3>Rule</h3>{{range .Rules}}<label><input type="checkbox" id="rule-{{urlquery .}}" name="rule" value="{{.}}" {{if selected $.Filters.rule .}}checked{{end}} onchange="this.form.elements.filter_open.value='more';this.form.requestSubmit()">{{.}}</label>{{end}}{{end}}<h3>Organization</h3>{{range .Organizations}}<label><input type="checkbox" id="org-{{urlquery .Login}}" name="org" value="{{.Login}}" {{if selected $.Filters.org .Login}}checked{{end}} onchange="this.form.elements.filter_open.value='more';this.form.requestSubmit()">{{.Login}}</label>{{end}}<h3>Language</h3>{{range .Languages}}<label><input type="checkbox" id="language-{{urlquery .}}" name="language" value="{{.}}" {{if selected $.Filters.language .}}checked{{end}} onchange="this.form.elements.filter_open.value='more';this.form.requestSubmit()">{{.}}</label>{{end}}<h3>Visibility</h3>{{range .Visibilities}}<label><input type="checkbox" id="visibility-{{urlquery .}}" name="visibility" value="{{.}}" {{if selected $.Filters.visibility .}}checked{{end}} onchange="this.form.elements.filter_open.value='more';this.form.requestSubmit()">{{.}}</label>{{end}}</div></details>
<a class="clear-filters" href="{{.ResetURL}}">Clear filters</a></div>
{{if .Chips}}<div class="filter-chips"><span>Showing</span>{{range .Chips}}<a href="{{.URL}}" aria-label="Remove {{.Label}}">{{.Label}} <span>×</span></a>{{end}}</div>{{end}}</form>

{{end}}
{{if eq .View "alerts"}}
<section class="panel"><div class="panel-head"><div><h2>Issues worth investigating</h2><p>Counts show matching / total before filters. Select an issue to see where it occurs.</p></div><span class="count">{{countOf (len .Groups) .TotalGroups}} issue groups · {{countOf .Stats.Open .Stats.Totals.Open}} occurrences</span></div><div class="table-wrap"><table class="issue-table"><thead><tr><th aria-sort="{{sortAria .SortColumn "rule" .SortDirection}}"><a href="{{index .SortLinks "rule"}}">Issue {{sortGlyph .SortColumn "rule" .SortDirection}}</a></th><th aria-sort="{{sortAria .SortColumn "severity" .SortDirection}}"><a href="{{index .SortLinks "severity"}}">Priority {{sortGlyph .SortColumn "severity" .SortDirection}}</a></th><th aria-sort="{{sortAria .SortColumn "repositories" .SortDirection}}"><a href="{{index .SortLinks "repositories"}}">Repositories {{sortGlyph .SortColumn "repositories" .SortDirection}}</a></th><th aria-sort="{{sortAria .SortColumn "alerts" .SortDirection}}"><a href="{{index .SortLinks "alerts"}}">Occurrences {{sortGlyph .SortColumn "alerts" .SortDirection}}</a></th><th aria-sort="{{sortAria .SortColumn "updated" .SortDirection}}"><a href="{{index .SortLinks "updated"}}">Alert updated {{sortGlyph .SortColumn "updated" .SortDirection}}</a></th></tr>
</thead>
<tbody>{{range .Groups}}<tr><td class="issue-cell"><a class="primary issue-title" href="{{.URL}}">{{.Name}}</a><span class="secondary-text">{{template "scanner-badge" .Tool}} <span class="mono">{{.RuleID}}</span></span>{{if and .Description (ne .Description .Name)}}<p class="row-description">{{.Description}}</p>{{end}}</td><td>{{template "group-priorities" .}}</td><td><a class="metric" href="{{.URL}}">{{countOf .RepositoryCount .Totals.AffectedRepositories}}</a><span class="secondary-text">affected repositories</span></td><td><a class="metric" href="{{.URL}}">{{countOf .AlertCount .Totals.Open}}</a><span class="secondary-text">open findings</span></td><td class="date-cell">{{.UpdatedAt.Format "Jan 2, 2006"}}<a class="secondary-text" href="{{.URL}}">Explore issue →</a></td></tr>
{{else}}<tr><td colspan="5" class="empty">No issues match this scope. <a href="{{.ResetURL}}">Clear filters to explore all priorities.</a></td></tr>
{{end}}</tbody></table></div></section>

{{else if eq .View "repositories"}}
<section class="panel"><div class="panel-head"><div><h2>Repository findings</h2><p>Counts show matching / total before filters. No open alerts does not establish scanning coverage.</p></div><span class="count">{{countOf (len .Repositories) .Stats.Totals.TotalRepositories}} repositories</span></div><div class="table-wrap"><table class="estate-table"><thead><tr><th aria-sort="{{sortAria .SortColumn "repository" .SortDirection}}"><a href="{{index .SortLinks "repository"}}">Repository {{sortGlyph .SortColumn "repository" .SortDirection}}</a></th><th aria-sort="{{sortAria .SortColumn "priority" .SortDirection}}"><a href="{{index .SortLinks "priority"}}">Priority breakdown {{sortGlyph .SortColumn "priority" .SortDirection}}</a></th><th aria-sort="{{sortAria .SortColumn "alerts" .SortDirection}}"><a href="{{index .SortLinks "alerts"}}">Open occurrences {{sortGlyph .SortColumn "alerts" .SortDirection}}</a></th><th>Explore</th></tr>
</thead>
<tbody>{{range .Repositories}}<tr><td><a class="primary issue-title" href="{{repoURL $ .ID}}">{{.Name}}</a><span class="secondary-text">{{.Org}} · {{if .Language}}{{.Language}} · {{end}}{{.Visibility}}</span><p class="row-description" title="{{.Description}}">{{.Description}}</p></td><td>{{template "repo-priorities" .}}</td><td><span class="metric {{if not .AlertCount}}zero{{end}}">{{countOf .AlertCount .Totals.Open}}</span>{{if not .AlertCount}}<span class="secondary-text">No matching open alerts</span>{{end}}</td><td><a href="{{repoURL $ .ID}}" aria-label="View repository alerts">View findings →</a></td></tr>
{{else}}<tr><td colspan="4" class="empty">No repositories match these filters. <a href="{{.ResetURL}}">Clear filters</a></td></tr>
{{end}}</tbody></table></div></section>

{{else if eq .View "repository"}}
<div class="repo-context"><span>{{.Repository.Visibility}} · {{if .Repository.Language}}{{.Repository.Language}}{{else}}Language not reported{{end}}</span><a href="{{.Repository.URL}}" target="_blank" rel="noopener">Repository on GitHub ↗</a><form method="post" action="{{.RefreshURL}}" hx-post="{{.RefreshURL}}" hx-target="body" hx-select="body" hx-config="timeout:0"><button class="btn secondary">Refresh repository</button></form>
</div>
<section class="panel"><div class="panel-head"><div><h2>Open findings in {{.Repository.Name}}</h2><p>Select a finding for its message, exact location, and scan context.</p></div><span class="count">{{countOf (len .Alerts) .Stats.Totals.Open}} occurrences</span></div>{{template "finding-table" .}}</section>

{{else if eq .View "rule"}}

<section class="panel"><div class="panel-head"><div><h2>Where this issue occurs</h2><p>Compare affected repositories before opening the individual findings.</p></div></div><div class="repository-impact">{{range .RuleRepositories}}<a href="{{repoURL $ .ID}}"><span><strong>{{.Name}}</strong><span class="secondary-text">{{.Org}}</span></span><span><b>{{countOf .AlertCount .Totals.Open}}</b> occurrences{{template "repo-priorities" .}}</span><span>→</span></a>{{else}}<div class="empty">No affected repositories in this filtered scope.</div>{{end}}</div></section>

<section class="panel"><div class="panel-head"><div><h2>Individual occurrences</h2><p>Each row is one GitHub alert. Open a finding to inspect its evidence.</p></div><span class="count">{{countOf (len .Alerts) .Stats.Totals.Open}} alerts</span></div>{{template "finding-table" .}}</section>

{{else if eq .View "alert"}}
<div class="occurrence-heading"><div><span class="severity severity-{{severityClass .Alert.Priority}}">{{.Alert.Priority}}</span> <span class="count">Open</span><span class="secondary-text">{{.Alert.Repository}} · GitHub alert #{{.Alert.GitHubID}}</span></div><a class="btn" href="{{.Alert.URL}}" target="_blank" rel="noopener">Open on GitHub ↗</a></div>
<div class="detail-grid"><div><section class="panel detail-section evidence"><div class="eyebrow">Reported evidence</div><h2>Finding message</h2>{{if .Alert.Message}}<div class="message">{{.Alert.Message}}</div>{{else}}<p class="description">The scanner did not provide a finding message.</p>{{end}}<h2 style="margin-top:24px">Affected location</h2>{{if .Alert.Path}}<a class="location" href="{{codeURL .Alert}}" target="_blank" rel="noopener"><strong>{{lineRange .Alert}}</strong><span class="secondary-text">View source at the reported revision ↗</span></a>{{else}}<p class="description">Location not reported by the scanner.</p>{{end}}<p class="secondary-text">{{if .Alert.Ref}}Ref: <span class="mono">{{.Alert.Ref}}</span>{{end}}{{if .Alert.CommitSHA}} · Commit: <span class="mono">{{shortsha .Alert.CommitSHA}}</span>{{end}}</p></section>
<section class="panel detail-section"><h2>About this issue</h2>{{if .Alert.RuleDescription}}<p class="description">{{.Alert.RuleDescription}}</p>{{else}}<p class="description">Rule description not provided by the scanner.</p>{{end}}<div class="tags">{{range .Alert.Tags}}<span class="tag">{{.}}</span>{{end}}</div><a class="related-link" href="{{ruleURL . .Alert.Tool .Alert.RuleID}}">Explore this issue across repositories →</a></section>
</div>
<aside><section class="panel detail-section"><h2>Finding context</h2><dl class="metadata"><div><dt>Repository</dt><dd><a href="{{repoURL . .Repository.ID}}">{{.Alert.Repository}}</a></dd></div><div><dt>Scanner</dt><dd>{{template "scanner-badge" .Alert.Tool}}{{if .Alert.ToolVersion}} <span class="secondary-text">Version {{.Alert.ToolVersion}}</span>{{end}}</dd></div><div><dt>Rule</dt><dd class="mono">{{.Alert.RuleID}}</dd></div><div><dt>First reported</dt><dd>{{.Alert.CreatedAt.Format "Jan 2, 2006 at 3:04 PM MST"}}</dd></div><div><dt>Alert updated</dt><dd>{{.Alert.UpdatedAt.Format "Jan 2, 2006 at 3:04 PM MST"}}</dd></div></dl></section>
<details class="panel detail-section scan-context"><summary>Scan details &amp; priority source</summary><dl class="metadata">{{if .Alert.ReportedSeverity}}<div><dt>Priority source</dt><dd>Reported severity: {{.Alert.ReportedSeverity}}{{if .Alert.SeveritySource}} (from {{.Alert.SeveritySource}}){{end}}</dd></div>{{end}}{{if .Alert.AnalysisKey}}<div><dt>Analysis key</dt><dd class="mono">{{.Alert.AnalysisKey}}</dd></div>{{end}}{{if .Alert.Environment}}<div><dt>Environment</dt><dd class="mono">{{.Alert.Environment}}</dd></div>{{end}}{{if .Alert.Category}}<div><dt>Category</dt><dd>{{.Alert.Category}}</dd></div>{{end}}{{if .Alert.CommitSHA}}<div><dt>Commit</dt><dd class="mono">{{.Alert.CommitSHA}}</dd></div>{{end}}</dl></details>
</aside></div>
{{else if eq .View "error"}}<div class="message">{{.ErrorMessage}}</div>{{end}}
</main>
{{define "scanner-badge"}}<span class="scanner scanner-{{scannerClass .}}">{{.}}</span>{{end}}
{{define "stats-priorities"}}<span class="severity severity-critical">Critical {{countOf .Critical .Totals.Critical}}</span><span class="severity severity-high">High {{countOf .High .Totals.High}}</span><span class="severity severity-medium">Medium {{countOf .Medium .Totals.Medium}}</span><span class="severity severity-low">Low {{countOf .Low .Totals.Low}}</span>{{if or .Other .Totals.Other}}<span class="severity severity-unknown">Unknown {{countOf .Other .Totals.Other}}</span>{{end}}{{end}}
{{define "repo-priorities"}}<div class="tags">{{if .Critical}}<span class="severity severity-critical">Critical {{countOf .Critical .Totals.Critical}}</span>{{end}}{{if .High}}<span class="severity severity-high">High {{countOf .High .Totals.High}}</span>{{end}}{{if .Medium}}<span class="severity severity-medium">Medium {{countOf .Medium .Totals.Medium}}</span>{{end}}{{if .Low}}<span class="severity severity-low">Low {{countOf .Low .Totals.Low}}</span>{{end}}{{if .Other}}<span class="severity severity-unknown">Unknown {{countOf .Other .Totals.Other}}</span>{{end}}</div>{{end}}
{{define "group-priorities"}}{{template "repo-priorities" .}}{{end}}
{{define "finding-table"}}<div class="table-wrap"><table class="finding-table"><thead><tr>{{if eq .View "rule"}}<th aria-sort="{{sortAria .SortColumn "repository" .SortDirection}}"><a href="{{index .AlertSortLinks "repository"}}">Repository {{sortGlyph .SortColumn "repository" .SortDirection}}</a></th>{{end}}<th>{{if eq .View "repository"}}<a href="{{index .SortLinks "rule"}}">Finding {{sortGlyph .SortColumn "rule" .SortDirection}}</a>{{else}}Finding evidence{{end}}</th><th>{{if eq .View "repository"}}<a href="{{index .SortLinks "severity"}}">Priority {{sortGlyph .SortColumn "severity" .SortDirection}}</a>{{else}}Priority{{end}}</th><th>{{if eq .View "rule"}}<a href="{{index .AlertSortLinks "updated"}}">Alert updated {{sortGlyph .SortColumn "updated" .SortDirection}}</a>{{else}}<a href="{{index .SortLinks "updated"}}">Alert updated {{sortGlyph .SortColumn "updated" .SortDirection}}</a>{{end}}</th></tr>
</thead>
<tbody>{{range .Alerts}}<tr>{{if eq $.View "rule"}}<td><a class="primary" href="{{repoURL $ .RepositoryID}}">{{.Repository}}</a><span class="secondary-text">GitHub alert #{{.GitHubID}}</span></td>{{end}}<td class="evidence-cell"><a class="primary issue-title" href="{{alertURL $ .ID}}">{{if eq $.View "repository"}}{{findingName .}}{{else}}{{if .Message}}{{.Message}}{{else}}Finding #{{.GitHubID}}{{end}}{{end}}</a>{{if and (eq $.View "repository") .Message}}<p class="row-description">{{.Message}}</p>{{end}}<span class="secondary-text mono">{{lineRange .}}</span><div class="finding-scanner">{{template "scanner-badge" .Tool}}{{if .Ref}}<span class="secondary-text mono">{{.Ref}}</span>{{end}}</div><a class="secondary-text" href="{{alertURL $ .ID}}" aria-label="View alert details">Inspect finding →</a></td><td><span class="severity severity-{{severityClass .Priority}}">{{.Priority}}</span></td><td class="date-cell">{{.UpdatedAt.Format "Jan 2, 2006"}}</td></tr>
{{else}}<tr><td colspan="4" class="empty">{{if eq .View "rule"}}No occurrences match these filters.{{else}}No alerts match these filters.{{end}} <a href="{{.ResetURL}}">Clear filters</a></td></tr>
{{end}}</tbody></table></div>{{end}}
<footer><p><a href="https://github.com/UnitVectorY-Labs">UnitVectorY Labs</a> | <a href="https://opensource.org/licenses/MIT">MIT License</a> | <a href="https://github.com/UnitVectorY-Labs/ghscanscope"><strong>ghscanscope</strong> on GitHub</a></p><p>Version: {{.Version}}</p></footer></body></html>`
