{{template "partials/header.tpl" .}}

{{with .Page.Event}}
<section class="wrap">
  <nav class="crumbs">
    <a href="/">Discover</a> <span>/</span> <a href="{{$.Page.BackURL}}">Events</a>
    <span>/</span> <span>Event details</span>
  </nav>

  <p class="back-line"><a class="back-link" href="{{$.Page.BackURL}}">&larr; Back to events</a></p>

  <div class="detail-grid">
    <div class="detail-main">
      <div class="detail-media">
        <img src="{{.ImageURL}}" alt="{{.Name}}">
      </div>

      {{if .Category}}<p class="detail-kicker">{{.Category}}</p>{{end}}
      <h1 class="detail-title">{{.Name}}</h1>

      {{if .HasDescription}}
        <h2 class="detail-subhead">About this event</h2>
        <p class="detail-body">{{.Description}}</p>
      {{else}}
        <h2 class="detail-subhead">About this event</h2>
        <p class="detail-body muted">No description was provided for this event.</p>
      {{end}}
    </div>

    <aside class="detail-side">
      <p class="eyebrow">Make a plan</p>
      <h2 class="side-title">The details</h2>

      <div class="side-row">
        <p class="side-label">When</p>
        <p class="side-value">{{.DateLabel}}</p>
        {{if .TimeLabel}}
          <p class="side-sub">{{.TimeLabel}}{{if .Timezone}} ({{.Timezone}}){{end}}</p>
        {{end}}
      </div>

      <div class="side-row">
        <p class="side-label">Where</p>
        <p class="side-value">{{if .VenueName}}{{.VenueName}}{{else}}Venue to be announced{{end}}</p>
        {{if .VenueAddress}}<p class="side-sub">{{.VenueAddress}}</p>{{end}}
        {{if .VenueLocation}}<p class="side-sub">{{.VenueLocation}}</p>{{end}}
      </div>

      {{if .Category}}
      <div class="side-row">
        <p class="side-label">Category</p>
        <p class="side-value">{{.Category}}</p>
      </div>
      {{end}}

      <!-- Points at our own route. The provider URL is resolved and validated
           on the server, then returned as a 302. -->
      <a class="btn btn-primary btn-block" href="{{.RedirectURL}}" rel="noopener">
        View tickets <span aria-hidden="true">&#8599;</span>
      </a>
      <p class="side-note">Opens the official ticket provider in a new step. The link is checked on our server first.</p>
    </aside>
  </div>
</section>
{{end}}

{{template "partials/footer.tpl" .}}