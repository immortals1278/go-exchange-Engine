import { useState } from 'react'
import './App.css'

function App() {
  const [activeTab, setActiveTab] = useState('trade')
  const [userID, setUserID] = useState(localStorage.getItem('userID') || '')
  const [isLoggedIn, setIsLoggedIn] = useState(!!localStorage.getItem('userID'))
  const [message, setMessage] = useState('')

  const handleLogin = async (e) => {
    e.preventDefault()
    if (!userID) {
      setMessage('❌ Please enter user ID')
      // 3秒后清除消息
      setTimeout(() => setMessage(''), 3000)
      return
    }

    try {
      // 调用真实的后端登录 API
      const response = await fetch('http://localhost:8080/api/login', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ user_id: userID })
      })
      
      const data = await response.json()
      if (data.code === 0) {
        setIsLoggedIn(true)
        // 存储 userID 到本地存储，以便在刷新页面后仍然保持登录状态
        localStorage.setItem('userID', userID)
        setMessage('✅ Login successful')
        // 3秒后清除消息
        setTimeout(() => setMessage(''), 3000)
      } else {
        setMessage('❌ ' + data.msg)
        // 3秒后清除消息
        setTimeout(() => setMessage(''), 3000)
      }
    } catch (error) {
      setMessage('❌ Login failed: ' + error.message)
      // 3秒后清除消息
      setTimeout(() => setMessage(''), 3000)
    }
  }

  return (
    <div className="app">
      <header className="header">
        <h1>Exchange Platform</h1>
        {!isLoggedIn ? (
          <form onSubmit={handleLogin} className="login-form">
            <input
              type="text"
              placeholder="Enter User ID"
              value={userID}
              onChange={(e) => setUserID(e.target.value)}
              className="login-input"
            />
            <button type="submit" className="login-button">Login</button>
          </form>
        ) : (
          <div className="user-info">
            <span>Welcome, {userID}</span>
            <button 
              className="logout-button"
              onClick={() => {
                setIsLoggedIn(false)
                localStorage.removeItem('userID')
                setUserID('')
              }}
            >
              Logout
            </button>
          </div>
        )}
      </header>

      {message && (
        <div className="message">{message}</div>
      )}

      {isLoggedIn && (
        <div className="main-content">
          <nav className="nav">
            <button
              className={`nav-button ${activeTab === 'trade' ? 'active' : ''}`}
              onClick={() => setActiveTab('trade')}
            >
              Trade
            </button>
            <button
              className={`nav-button ${activeTab === 'balance' ? 'active' : ''}`}
              onClick={() => setActiveTab('balance')}
            >
              Balances
            </button>
          </nav>

          <div className="content">
            {activeTab === 'trade' && <TradePage userID={userID} setMessage={setMessage} />}
            {activeTab === 'balance' && <BalancePage userID={userID} setMessage={setMessage} />}
          </div>
        </div>
      )}
    </div>
  )
}

function TradePage({ userID, setMessage }) {
  const [symbol, setSymbol] = useState('BTC/USDT')
  const [side, setSide] = useState('buy')
  const [price, setPrice] = useState('50000')
  const [quantity, setQuantity] = useState('0.01')
  const [balances, setBalances] = useState({})

  const fetchBalances = async () => {
    try {
      // 这里我们可以通过下单或取消订单的响应获取余额
      // 暂时模拟数据
      setBalances({ USDT: '10000', BTC: '0' })
    } catch (error) {
      setMessage('Failed to fetch balances: ' + error.message)
    }
  }

  const handlePlaceOrder = async (e) => {
    e.preventDefault()
    
    try {
      const response = await fetch('http://localhost:8080/api/order', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          user_id: userID,
          symbol: symbol,
          side: side,
          price: parseFloat(price),
          quantity: parseFloat(quantity)
        })
      })
      
      const data = await response.json()
      
      if (data.code === 0) {
        setMessage('✅ Order placed successfully')
        setBalances(data.data)
      } else {
        setMessage('❌ Order failed: ' + data.msg)
      }
    } catch (error) {
      console.error('Error:', error)
      setMessage('❌ Order failed: ' + error.message)
    } finally {
      // 无论成功与否，都清空价格和数量输入框
      setPrice('')
      setQuantity('')
      // 3秒后清除消息
      setTimeout(() => setMessage(''), 3000)
    }
  }

  return (
    <div className="trade-page">
      <h2>Trade</h2>
      <div className="balance-info">
        <h3>Balances</h3>
        <div className="balance-item">USDT: {balances.USDT || '0'}</div>
        <div className="balance-item">BTC: {balances.BTC || '0'}</div>
      </div>
      <form onSubmit={handlePlaceOrder} className="order-form">
        <div className="form-group">
          <label>Symbol</label>
          <select 
            value={symbol} 
            onChange={(e) => setSymbol(e.target.value)}
            className="form-input"
          >
            <option value="BTC/USDT">BTC/USDT</option>
          </select>
        </div>
        <div className="form-group">
          <label>Side</label>
          <div className="side-buttons">
            <button 
              type="button"
              className={`side-button ${side === 'buy' ? 'active' : ''}`}
              onClick={() => setSide('buy')}
            >
              BUY
            </button>
            <button 
              type="button"
              className={`side-button ${side === 'sell' ? 'active' : ''}`}
              onClick={() => setSide('sell')}
            >
              SELL
            </button>
          </div>
        </div>
        <div className="form-group">
          <label>Price (USDT)</label>
          <input
            type="number"
            value={price}
            onChange={(e) => setPrice(e.target.value)}
            className="form-input"
            step="0.01"
          />
        </div>
        <div className="form-group">
          <label>Quantity</label>
          <input
            type="number"
            value={quantity}
            onChange={(e) => setQuantity(e.target.value)}
            className="form-input"
            step="0.0001"
          />
        </div>
        <button type="submit" className="place-order-button">
          Place Order
        </button>
      </form>
    </div>
  )
}

function BalancePage({ userID, setMessage }) {
  const [balances, setBalances] = useState({ USDT: '10000', BTC: '0.00' })

  const fetchBalances = async () => {
    try {
      // 使用真实的 API 调用获取余额
      const response = await fetch(`http://localhost:8080/api/balance?user_id=${userID}`)
      const data = await response.json()
      if (data.code === 0) {
        setBalances(data.data)
        setMessage('✅ Balances refreshed successfully')
        // 3秒后清除消息
        setTimeout(() => setMessage(''), 3000)
      } else {
        setMessage('❌ Failed to fetch balances: ' + data.msg)
        // 3秒后清除消息
        setTimeout(() => setMessage(''), 3000)
      }
    } catch (error) {
      setMessage('❌ Failed to fetch balances: ' + error.message)
      // 3秒后清除消息
      setTimeout(() => setMessage(''), 3000)
    }
  }

  return (
    <div className="balance-page">
      <h2>Balances</h2>
      <div className="balance-list">
        <div className="balance-card">
          <h3>USDT</h3>
          <div className="balance-amount">{balances.USDT || '0'}</div>
          <div className="balance-label">Available</div>
        </div>
        <div className="balance-card">
          <h3>BTC</h3>
          <div className="balance-amount">{balances.BTC || '0'}</div>
          <div className="balance-label">Available</div>
        </div>
      </div>
      <button 
        className="refresh-button"
        onClick={fetchBalances}
      >
        Refresh Balances
      </button>
    </div>
  )
}

export default App
