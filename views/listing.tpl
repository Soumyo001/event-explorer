{{template "partials/header.tpl" .}}

<section class="wrap listing-head">
  <nav class="crumbs">
    <a href="/">Discover</a> <span>/</span> <span>{{.Page.City}}</span>
  </nav>
  <p class="eyebrow">Your city, your plans</p>
  <div class="listing-title-row">
    <h1 class="hero-title">What is on in <span class="accent">{{.Page.City}}</span>.</h1>
    <a class="btn btn-ghost" href="/">Change city <span aria-hidden="true">&#8599;</span></a>
  </div>
  <p class="hero-sub">Music and sports, together. Find your next reason to go out.</p>
  <p class="meta-line">
    <span><strong>Location</strong> {{.Page.City}}, {{.Page.CountryCode}}</span>
    <span><strong>Data</strong> Ticketmaster Discovery</span>
  </p>
</section>

{{if .Page.AllFailed}}
<section class="wrap">
  <div class="notice notice-error">
    <h2 class="notice-title">Events could not be loaded</h2>
    <p>The event service is not responding right now. Please try again in a moment.</p>
    <p><a class="btn btn-primary" href="/">&larr; Choose another city</a></p>
  </div>
</section>
{{else}}
  {{range .Page.Sections}}
  <section class="wrap section-block">
    <div class="section-head">
      <h2 class="section-title">{{.Title}} <span class="count">{{.Count}}</span></h2>
      {{if .Cached}}<span class="badge">From cache</span>{{end}}
    </div>

    {{if .HasError}}
      <div class="notice notice-error">
        <p>{{.Err}}</p>
      </div>
    {{else if .IsEmpty}}
      <div class="notice">
        <p>No {{.Title}} events found in this city right now.</p>
      </div>
    {{else}}
      <ul class="card-grid">
        {{range .Events}}
        <li class="card">
          <a class="card-media" href="{{.DetailsURL}}">
            <img src="{{.ImageURL}}" alt="{{.Name}}" loading="lazy">
          </a>
          <div class="card-body">
            <p class="card-date">{{.DateLabel}}{{if .TimeLabel}} &middot; {{.TimeLabel}}{{end}}</p>
            <h3 class="card-title"><a href="{{.DetailsURL}}">{{.Name}}</a></h3>
            {{if .VenueName}}<p class="card-venue">{{.VenueName}}</p>{{end}}
            <div class="card-foot">
              <span class="card-place">{{.VenueLocation}}</span>
              <a class="card-link" href="{{.DetailsURL}}">View details <span aria-hidden="true">&#8599;</span></a>
            </div>
          </div>
        </li>
        {{end}}
      </ul>
    {{end}}
  </section>
  {{end}}
{{end}}

{{template "partials/footer.tpl" .}}