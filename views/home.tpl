{{template "partials/header.tpl" .}}

<section class="hero wrap">
  <p class="eyebrow">Less scrolling. More going.</p>
  <h1 class="hero-title">
    A city of possibilities.<br>
    <span class="accent">Find your next one.</span>
  </h1>
  <p class="hero-sub">Discover music and sports in one place.</p>
  <p class="hero-sub">Choose your city. Find something worth heading out for.</p>
  <ul class="pill-row">
    <li class="pill">Live music</li>
    <li class="pill">Sports &amp; matchdays</li>
    <li class="pill">One simple search</li>
  </ul>
</section>

<section class="wrap">
  <div class="panel">
    <h2 class="panel-title">
      Where are we going?
      <span class="panel-hint">Start with a city, then explore what is on.</span>
    </h2>
    
    <form class="search" id="city-search" action="/events" method="get" novalidate>
      <label class="field-label" for="city-input">Choose a city</label>
      <div class="search-row">
        <div class="search-field">
          <input type="text" id="city-input" name="cityInput"
                 class="search-input" autocomplete="off" spellcheck="false"
                 placeholder="Search a city, e.g. Toronto"
                 aria-controls="suggestions" aria-expanded="false"
                 aria-autocomplete="list" role="combobox">
          <ul class="suggestions" id="suggestions" role="listbox" hidden></ul>
        </div>
        <button type="submit" class="btn btn-primary" id="search-button" disabled>
          Explore events <span aria-hidden="true">&rarr;</span>
        </button>
      </div>

      <input type="hidden" name="city" id="city-value">
      <input type="hidden" name="countryCode" id="country-value">

      <p class="field-note">
        <span id="search-hint">Type at least 3 characters and select a suggestion.</span>
        <span class="attribution">{{.Page.Attribution}}</span>
      </p>
      <p class="field-error" id="search-error" role="alert" hidden></p>
    </form>
  </div>
</section>

{{if .Page.HasSampleCities}}
<section class="wrap">
  <div class="sample-box">
    <div class="sample-copy">
      <h3 class="sample-title">Ready-to-test sample cities</h3>
      <p class="sample-sub">Verified against the Ticketmaster catalogue.</p>
    </div>
    <ul class="sample-links">
      {{range .Page.SampleCities}}
      <li>
        <a class="chip" href="{{.ListingURL}}" title="{{.Note}}">
          {{.DisplayLabel}}{{if not .HasEvents}} &mdash; empty{{end}}
          <span aria-hidden="true">&#8599;</span>
        </a>
      </li>
      {{end}}
    </ul>
  </div>
</section>
{{end}}

<script src="/static/js/autocomplete.js" defer></script>

{{template "partials/footer.tpl" .}}