import React from 'react'
import {createRoot} from 'react-dom/client'
import App from './App'
import { getCurrentLanguage } from './locales/useI18n'

function ErrorScreen({error, title = 'AI Console error'}) {
    const message = error?.stack || error?.message || String(error || 'Unknown startup error')
    return (
        <main style={{
            minHeight: '100vh',
            boxSizing: 'border-box',
            padding: 24,
            background: '#050505',
            color: '#fff',
            fontFamily: 'Menlo, Monaco, monospace',
            whiteSpace: 'pre-wrap',
        }}>
            <h1 style={{fontSize: 18, margin: '0 0 12px'}}>{title}</h1>
            <pre style={{fontSize: 12, lineHeight: 1.5, overflow: 'auto'}}>{message}</pre>
        </main>
    )
}

class RootErrorBoundary extends React.Component {
    constructor(props) {
        super(props)
        this.state = {error: null}
    }

    static getDerivedStateFromError(error) {
        return {error}
    }

    componentDidCatch(error, info) {
        console.error('[AI Console render]', error, info)
    }

    render() {
        if (this.state.error) {
            return <ErrorScreen error={this.state.error} title="AI Console render error"/>
        }
        return this.props.children
    }
}

const container = document.getElementById('root')
const root = createRoot(container)

try {
    const startupLang = getCurrentLanguage()
    document.documentElement.lang = startupLang
    root.render(
        <React.StrictMode>
            <RootErrorBoundary>
                <App/>
            </RootErrorBoundary>
        </React.StrictMode>
    )
} catch (error) {
    console.error('[AI Console startup]', error)
    root.render(<ErrorScreen error={error} title="AI Console startup error"/>)
}
