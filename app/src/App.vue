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
  name: 'app',
  mounted: function() {
    document.title = "quice - select user";
  }
}
</script>
<!-- styling for the component -->
<style>
html {
  background-color: #222222;
}
</style>
