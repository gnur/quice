<template>
  <section class="is-primary is-medium">
    <div class="section">
      <div class="container">
        <router-view></router-view>
      </div>
    </div>
  </section>
</template>

<script>
export default {
  name: "app",
  mounted: function() {
    document.title = "quice - select user";
  }
};
</script>
<!-- styling for the component -->
<style lang="scss">
@charset "utf-8";

// Import Bulma's core
@import "~bulma/sass/utilities/_all";

/// Custom theme here

// Import Bulma and Buefy styles
@import "~bulma";
@import "~buefy/src/scss/buefy";

// Import icons
@import "~@mdi/font/css/materialdesignicons.min.css";
html {
  background-color: #222222;
}
</style>
