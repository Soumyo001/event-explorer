{{template "partials/header.tpl" .}}

<section class="wrap state-page">
  <p class="eyebrow">Error {{.Page.StatusCode}}</p>
  <h1 class="state-title">{{.Page.Heading}}</h1>
  <p class="state-message">{{.Page.Message}}</p>
  <p><a class="btn btn-primary" href="{{.Page.BackURL}}">&larr; Go back</a></p>
</section>

{{template "partials/footer.tpl" .}}