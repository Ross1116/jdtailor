import React from 'react'
import {createRoot} from 'react-dom/client'
import './style.css'
import App from './App'

class ErrorBoundary extends React.Component<{children: React.ReactNode}, {error: string}> {
  constructor(props: {children: React.ReactNode}) {
    super(props);
    this.state = {error: ''};
  }
  static getDerivedStateFromError(error: Error) {
    return {error: error.message + '\n' + (error.stack || '')};
  }
  render() {
    if (this.state.error) {
      return (
        <div style={{padding: 20, fontFamily: 'monospace', fontSize: 12, background: '#fee', color: '#900'}}>
          <h2>Render Error</h2>
          <pre style={{whiteSpace: 'pre-wrap'}}>{this.state.error}</pre>
        </div>
      );
    }
    return this.props.children;
  }
}

const container = document.getElementById('root')
if (!container) {
	throw new Error('Root container element not found')
}
const root = createRoot(container)

root.render(
    <React.StrictMode>
      <ErrorBoundary>
        <App/>
      </ErrorBoundary>
    </React.StrictMode>
)
