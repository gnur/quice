<template>
  <div class="tile is-ancestor is-10">

    <div v-for="user in users" :key="user" class="tile is-parent">
      <router-link :to="{ name: 'PlaylistSelect', params: { user: user }}" class="tile is-child box">
          <p class="title">{{ user }}</p>
        </router-link>
      </div>

  </div>
</template>

<script>
import axios from "axios";

export default {
  name: "UserSelect",
  data: function() {
    return {
      users: []
    };
  },
  methods: {
    getUsers: function() {
      axios.get("/api/users").then(
        response => {
          this.users = response.data.users;
          console.log(this.users[0]);
        },
        error => {
          console.log(error);
        }
      );
    }
  },
  mounted: function() {
    this.getUsers();
  }
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped>
h1,
h2 {
  font-weight: normal;
}
ul {
  list-style-type: none;
  padding: 0;
}
li {
  display: inline-block;
  margin: 0 10px;
}
a {
  color: #42b983;
}
</style>
