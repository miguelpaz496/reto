<template>
  <div class="pagina">
    
    <div>
      <h1 class="title">Reto </h1>
    </div>
    <hr>
    
    <div>
      <b-container fluid>
        <br><br>
        <b-row class="text-center">
          <b-col cols="3" offset-md="2">
            <div class="field">
              <label class="label">Digite el dominio</label>
              <b-form-input name="num1" v-model="num1"  class="input" type="text"></b-form-input>
              <br>
              <b-button variant="primary" v-on:click="postreq()">Buscar</b-button>
              <br><br><br>
              <label class="label">Para ver los dominios consultados anteriormente</label>
              <b-button variant="primary" v-on:click="consulta()">Consulta</b-button>
              <br>
              <br>
              <br>
              <b-button variant="primary" v-on:click="limpiar()">Limpiar</b-button>
            </div>
          </b-col>
          <b-col cols="5" >
            <b-form-textarea
              id="textarea-small"
              v-model="respuesta"
              rows="12"
              max-rows="12"
            ></b-form-textarea>

          </b-col>

          <b-col cols="2" ></b-col>

        </b-row>
        <br><br>
        <b-row>
          <b-col sm="8" offset-md="2">
            
          </b-col>
          <b-col sm="2">
          </b-col>
        </b-row>
      </b-container>
    </div>


    <div>
       
    </div>

    <hr>

  </div>
</template>

<script>

import axios from './../../node_modules/axios';


export default {
  name: 'Pagina',
  data: function() {
    return {
      num1: "",
      respuesta: "",
      respuesta2: ""
    }
  },
  methods: {
    postreq: function() {
      var infor = {"name": this.num1}
      this.respuesta = ""
      /*eslint-disable*/
      console.log(infor) 
      /*eslint-enable*/
      axios({ method: "POST", url: "http://127.0.0.1:8090", data: infor, headers: {"content-type": "text/plain" } }).then(result => { 
          // this.response = result.data;
          /*eslint-disable*/
          console.log(result);
          var obj = { items: result.data };
          var myJSON = JSON.stringify(obj);
          this.respuesta = obj;
          /*eslint-enable*/
        }).catch( error => {
            /*eslint-disable*/
            console.error(error);
            /*eslint-enable*/
      });
    },

    consulta: function() {
      var infor = {"name": this.num1}
      /*eslint-disable*/
      this.respuesta = ""
      console.log(infor) 
      /*eslint-enable*/
      axios({ method: "POST", url: "http://127.0.0.1:8090/informacion", data: infor, headers: {"content-type": "text/plain" } }).then(result => { 
          // this.response = result.data;
          /*eslint-disable*/
          console.log(result);
          var obj = { items: result.data };
          var myJSON = JSON.stringify(obj);
          this.respuesta = myJSON;
          /*eslint-enable*/
        }).catch( error => {
            /*eslint-disable*/
            console.error(error);
            /*eslint-enable*/
      });
    },

      limpiar: function() {

      this.respuesta = ""

    },

  }
}
</script>