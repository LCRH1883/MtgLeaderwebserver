package httpapi

import (
	"html/template"
	"net/http"
)

const (
	androidTestingURL = "https://play.google.com/apps/testing/com.intagri.mtgleader"
	androidStoreURL   = "https://play.google.com/store/apps/details?id=com.intagri.mtgleader"
	privacyUpdated    = "2025-12-29"
)

var publicPageT = template.Must(template.New("public").Parse(publicLayout))

type publicPageData struct {
	Title string
	Body  template.HTML
}

func (a *api) handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	renderPublicPage(w, http.StatusOK, "MTG Leader", publicHomeBody)
}

func (a *api) handlePrivacyWeb(w http.ResponseWriter, r *http.Request) {
	renderPublicPage(w, http.StatusOK, "Web Privacy Policy", publicPrivacyWebBody)
}

func (a *api) handlePrivacyAndroid(w http.ResponseWriter, r *http.Request) {
	renderPublicPage(w, http.StatusOK, "Android Privacy Policy", publicPrivacyAndroidBody)
}

func renderPublicPage(w http.ResponseWriter, status int, title string, body template.HTML) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = publicPageT.Execute(w, publicPageData{
		Title: title,
		Body:  body,
	})
}

const publicLayout = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width,initial-scale=1" />
    <title>{{.Title}}</title>
    <link rel="preconnect" href="https://fonts.googleapis.com" />
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
    <link href="https://fonts.googleapis.com/css2?family=Space+Grotesk:wght@500;700&family=Work+Sans:wght@400;500;600&display=swap" rel="stylesheet" />
    <style>
      :root{
        --bg:#0b0b0f;
        --bg-2:#11121a;
        --ink:#f8fafc;
        --muted:#cbd5f5;
        --accent:#ef4444;
        --accent-2:#f97316;
        --card:rgba(15,23,42,0.85);
        --line:rgba(148,163,184,0.25);
        --shadow:0 18px 40px rgba(2,6,23,0.6);
        color-scheme:dark;
      }
      *{box-sizing:border-box}
      body{
        margin:0;
        font-family:"Work Sans","Helvetica Neue",Arial,sans-serif;
        color:var(--ink);
        background:var(--bg);
        min-height:100vh;
      }
      body::before{
        content:"";
        position:fixed;
        width:380px;
        height:380px;
        right:-140px;
        top:-160px;
        background:radial-gradient(circle,#fb7185 0%,rgba(251,113,133,0.22) 60%,rgba(251,113,133,0) 70%);
        z-index:-1;
      }
      body::after{
        content:"";
        position:fixed;
        width:420px;
        height:420px;
        left:-180px;
        bottom:-160px;
        background:radial-gradient(circle,#fca5a5 0%,rgba(252,165,165,0.2) 60%,rgba(252,165,165,0) 70%);
        z-index:-1;
      }
      header{
        display:flex;
        align-items:center;
        justify-content:space-between;
        gap:16px;
        padding:24px clamp(20px,4vw,64px);
      }
      .logo{
        display:flex;
        align-items:center;
        gap:14px;
        font-family:"Space Grotesk","Work Sans",sans-serif;
        text-decoration:none;
        color:inherit;
      }
      .logo-mark{
        width:46px;
        height:46px;
        border-radius:14px;
        display:flex;
        align-items:center;
        justify-content:center;
        font-weight:700;
        letter-spacing:1px;
        color:white;
        background:linear-gradient(135deg,var(--accent),var(--accent-2));
      }
      .logo-title{
        font-weight:700;
        font-size:18px;
      }
      .logo-sub{
        font-size:12px;
        color:var(--muted);
      }
      .nav{
        display:flex;
        gap:10px;
        flex-wrap:wrap;
      }
      .nav a{
        text-decoration:none;
        font-weight:600;
        font-size:13px;
        padding:8px 14px;
        border-radius:999px;
        border:1px solid var(--line);
        background:var(--card);
        color:var(--ink);
      }
      .nav a.primary{
        background:var(--accent);
        border-color:var(--accent);
        color:white;
        box-shadow:0 12px 24px rgba(185,28,28,0.2);
      }
      main{
        max-width:1120px;
        margin:0 auto;
        padding:0 clamp(20px,4vw,64px) 80px;
      }
      h1,h2{
        font-family:"Space Grotesk","Work Sans",sans-serif;
        margin:0 0 12px;
      }
      .hero{
        display:grid;
        grid-template-columns:minmax(0,1.1fr) minmax(0,0.9fr);
        gap:32px;
        margin-top:24px;
      }
      .badge{
        display:inline-flex;
        align-items:center;
        gap:8px;
        padding:6px 12px;
        border-radius:999px;
        border:1px solid rgba(185,28,28,0.2);
        background:rgba(185,28,28,0.1);
        color:var(--accent);
        font-size:12px;
        font-weight:600;
        letter-spacing:0.4px;
        text-transform:uppercase;
      }
      .lead{
        color:var(--muted);
        line-height:1.6;
        margin:0 0 16px;
      }
      .cta{
        display:flex;
        flex-wrap:wrap;
        gap:12px;
        margin-top:20px;
      }
      .button{
        display:inline-flex;
        align-items:center;
        justify-content:center;
        padding:12px 18px;
        border-radius:12px;
        border:1px solid var(--accent);
        background:var(--accent);
        color:white;
        text-decoration:none;
        font-weight:600;
        box-shadow:0 14px 24px rgba(185,28,28,0.2);
      }
      .button.ghost{
        background:var(--card);
        color:var(--accent);
        border-color:rgba(185,28,28,0.3);
        box-shadow:none;
      }
      .card{
        background:var(--card);
        border:1px solid var(--line);
        border-radius:18px;
        padding:18px;
        box-shadow:var(--shadow);
      }
      .grid{
        display:grid;
        grid-template-columns:repeat(auto-fit,minmax(220px,1fr));
        gap:16px;
        margin-top:24px;
      }
      .note{
        margin-top:20px;
        padding:16px;
        border-radius:16px;
        border:1px solid rgba(185,28,28,0.2);
        background:rgba(185,28,28,0.08);
        font-weight:600;
      }
      .link-row{
        display:flex;
        flex-wrap:wrap;
        gap:12px;
        margin-top:16px;
      }
      .link-row a{
        color:var(--accent);
        font-weight:600;
        text-decoration:none;
      }
      footer{
        margin-top:36px;
        padding-top:18px;
        border-top:1px solid var(--line);
        color:var(--muted);
        font-size:13px;
        display:flex;
        flex-wrap:wrap;
        gap:12px;
        align-items:center;
        justify-content:space-between;
      }
      @media (max-width:900px){
        .hero{grid-template-columns:1fr}
        header{flex-direction:column;align-items:flex-start}
      }
    </style>
    <script>
      (() => {
        const root = document.documentElement;
        const storageKey = "mtg-theme";
        try {
          const stored = localStorage.getItem(storageKey);
          if (stored === "dark" || stored === "light") {
            root.setAttribute("data-theme", stored);
          }
        } catch (err) {}
      })();
    </script>
  </head>
  <body>
    <header>
      <a class="logo" href="/">
        <span class="logo-mark">MTG</span>
        <span>
          <div class="logo-title">MTG Leader</div>
          <div class="logo-sub">Magic: The Gathering playgroup toolkit</div>
        </span>
      </a>
      <nav class="nav">
        <a class="primary" href="/app/login">User Login</a>
      </nav>
    </header>
    <main>
      {{.Body}}
      <footer>
        <div>Copyright 2025 MTG Leader. Built for Magic: The Gathering communities.</div>
        <div class="link-row">
          <a href="/privacy">Web privacy</a>
          <a href="/privacy/android">Android privacy</a>
          <a href="/app/login">User login</a>
        </div>
      </footer>
    </main>
  </body>
</html>`

var publicHomeBody = template.HTML(`
<section class="hero">
  <div>
    <span class="badge">In development</span>
    <h1>MTG Leader is building the next-gen hub for Magic: The Gathering pods.</h1>
    <p class="lead">Track matches, manage friends, and surface stats that matter for your playgroup. The Android app is in active development now, and iOS work will begin once Android is complete.</p>
    <div class="cta">
      <a class="button" href="/app/login">User Login</a>
      <a class="button ghost" href="` + androidTestingURL + `" target="_blank" rel="noopener">Android Test</a>
      <a class="button ghost" href="` + androidStoreURL + `" target="_blank" rel="noopener">Play Store</a>
    </div>
    <div class="note">Android is first in development. iOS work starts after Android is completed.</div>
  </div>
  <div class="card">
    <h2>What is MTG Leader?</h2>
    <p class="lead">A focused app for Magic: The Gathering players who want to keep track of their pod, track match outcomes, and see who is on top. We are shipping the Android experience first, then expanding to iOS with the same backend.</p>
    <div class="grid">
      <div class="card">
        <h2>Pod management</h2>
        <p class="lead">Add friends, track who is in your regular group, and keep everyone connected.</p>
      </div>
      <div class="card">
        <h2>Match tracking</h2>
        <p class="lead">Capture results, placements, and formats to build a shared record.</p>
      </div>
      <div class="card">
        <h2>Stats overview</h2>
        <p class="lead">See win rates and trends across Commander, Brawl, Standard, and Modern.</p>
      </div>
    </div>
  </div>
</section>
`)

var publicPrivacyWebBody = template.HTML(`
<section class="card">
  <span class="badge">Web Privacy Policy</span>
  <h1>MTG Leader Web Privacy Policy</h1>
  <p class="lead">Last updated: ` + privacyUpdated + `</p>
  <p class="lead">MTG Leader is in development for Magic: The Gathering players. This policy describes how the web experience handles data.</p>
  <h2>Data we collect</h2>
  <ul>
    <li>Account information such as email, username, and password (stored as a secure hash).</li>
    <li>Profile details you provide, like display name and avatar.</li>
    <li>Gameplay data such as matches, placements, friends, and stats.</li>
    <li>Session and security data, including cookies needed to keep you signed in.</li>
  </ul>
  <h2>How we use data</h2>
  <ul>
    <li>To authenticate your account and keep you signed in.</li>
    <li>To power app features like matches, friends, and stats.</li>
    <li>To protect the service and prevent abuse.</li>
  </ul>
  <h2>Sharing</h2>
  <p class="lead">We do not sell your personal data. We share data only with infrastructure providers needed to run the service.</p>
  <h2>Retention</h2>
  <p class="lead">We keep data for as long as your account is active or as required for service operation. You can request deletion.</p>
  <h2>Your choices</h2>
  <p class="lead">You can update profile information in the app. For deletion requests, use the support contact listed in the app or store listing.</p>
</section>
`)

var publicPrivacyAndroidBody = template.HTML(`
<section class="card">
  <span class="badge">Android Privacy Policy</span>
  <h1>MTG Leader Android Privacy Policy</h1>
  <p class="lead">Last updated: ` + privacyUpdated + `</p>
  <p class="lead">The Android app connects to the same MTG Leader backend as the web experience. Android is in active development and ships first.</p>
  <h2>Data we collect</h2>
  <ul>
    <li>Account data such as email, username, and password (stored as a secure hash).</li>
    <li>Profile data like display name and avatar.</li>
    <li>Gameplay data including matches, placements, friends, and stats.</li>
    <li>Operational diagnostics provided by Google Play reporting tools.</li>
  </ul>
  <h2>How we use data</h2>
  <ul>
    <li>To run core features like login, friends, and match tracking.</li>
    <li>To improve app stability and performance.</li>
    <li>To secure the service and prevent abuse.</li>
  </ul>
  <h2>Sharing</h2>
  <p class="lead">We do not sell personal data. Data is shared only with infrastructure providers and Google Play services required to deliver the app.</p>
  <h2>Retention</h2>
  <p class="lead">Data is retained while your account is active or as required for service operations. You can request deletion.</p>
  <h2>Your choices</h2>
  <p class="lead">You can update profile information in the app. For deletion requests, use the support contact listed in the Play Store entry.</p>
</section>
`)
