import * as App from './bindings/naraberu/app.js';

window.naraberuService = App;
window.dispatchEvent(new Event('naraberu-service-ready'));
