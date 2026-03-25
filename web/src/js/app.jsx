import { h, render } from 'preact';
import Router from 'preact-router';
import { Layout } from './components/Layout.jsx';
import { LoginPage } from './pages/LoginPage.jsx';
import { IndexPage } from './pages/IndexPage.jsx';
import { DetailPage } from './pages/DetailPage.jsx';
import { isAuthenticated } from './store.js';

function App() {
  return (
    <Layout>
      <Router>
        <LoginPage path="/login" />
        <IndexPage path="/" />
        <DetailPage path="/urls/new" />
        <DetailPage path="/urls/:id" />
      </Router>
    </Layout>
  );
}

render(<App />, document.getElementById('app'));
