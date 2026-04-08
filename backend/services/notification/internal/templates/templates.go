package templates

import (
	"bytes"
	"fmt"
	"html/template"
)

var orderConfirmationTmpl = template.Must(template.New("order_confirmation").Parse(`
<!DOCTYPE html>
<html>
<body>
<h2>ご注文ありがとうございます</h2>
<p>{{.BuyerName}} 様</p>
<p>ご注文を承りました。</p>
<table>
  <tr><td><strong>注文番号:</strong></td><td>{{.OrderID}}</td></tr>
  <tr><td><strong>出品者:</strong></td><td>{{.SellerName}}</td></tr>
  <tr><td><strong>合計金額:</strong></td><td>¥{{.TotalAmount}}</td></tr>
</table>
<p>ご注文の詳細は、マイページからご確認いただけます。</p>
</body>
</html>
`))

var orderPaidTmpl = template.Must(template.New("order_paid").Parse(`
<!DOCTYPE html>
<html>
<body>
<h2>入金確認のお知らせ</h2>
<p>{{.SellerName}} 様</p>
<p>以下の注文の入金が確認されました。</p>
<table>
  <tr><td><strong>注文番号:</strong></td><td>{{.OrderID}}</td></tr>
  <tr><td><strong>金額:</strong></td><td>¥{{.TotalAmount}}</td></tr>
</table>
<p>商品の発送手続きをお願いいたします。</p>
</body>
</html>
`))

var orderShippedTmpl = template.Must(template.New("order_shipped").Parse(`
<!DOCTYPE html>
<html>
<body>
<h2>商品発送のお知らせ</h2>
<p>{{.BuyerName}} 様</p>
<p>ご注文の商品が発送されました。</p>
<table>
  <tr><td><strong>注文番号:</strong></td><td>{{.OrderID}}</td></tr>
</table>
<p>お届けまでしばらくお待ちください。</p>
</body>
</html>
`))

var lowStockAlertTmpl = template.Must(template.New("low_stock").Parse(`
<!DOCTYPE html>
<html>
<body>
<h2>在庫不足アラート</h2>
<p>以下の商品の在庫が基準値を下回りました。</p>
<table>
  <tr><td><strong>SKUコード:</strong></td><td>{{.SKUCode}}</td></tr>
  <tr><td><strong>商品名:</strong></td><td>{{.ProductName}}</td></tr>
  <tr><td><strong>現在庫数:</strong></td><td>{{.CurrentStock}}</td></tr>
  <tr><td><strong>基準値:</strong></td><td>{{.Threshold}}</td></tr>
</table>
<p>在庫の補充をご検討ください。</p>
</body>
</html>
`))

// OrderConfirmation generates a buyer order confirmation email.
func OrderConfirmation(orderID, buyerName string, totalAmount int64, sellerName string) (subject, body string) {
	subject = fmt.Sprintf("【ご注文確認】注文番号: %s", orderID)
	data := map[string]any{
		"OrderID":     orderID,
		"BuyerName":   buyerName,
		"TotalAmount": totalAmount,
		"SellerName":  sellerName,
	}
	body = render(orderConfirmationTmpl, data)
	return subject, body
}

// OrderPaidNotification generates a seller payment notification email.
func OrderPaidNotification(orderID, sellerName string, totalAmount int64) (subject, body string) {
	subject = fmt.Sprintf("【入金確認】注文番号: %s", orderID)
	data := map[string]any{
		"OrderID":     orderID,
		"SellerName":  sellerName,
		"TotalAmount": totalAmount,
	}
	body = render(orderPaidTmpl, data)
	return subject, body
}

// OrderShippedNotification generates a buyer shipping notification email.
func OrderShippedNotification(orderID, buyerName string) (subject, body string) {
	subject = fmt.Sprintf("【発送完了】注文番号: %s", orderID)
	data := map[string]any{
		"OrderID":   orderID,
		"BuyerName": buyerName,
	}
	body = render(orderShippedTmpl, data)
	return subject, body
}

// LowStockAlert generates a seller low stock alert email.
func LowStockAlert(skuCode, productName string, currentStock, threshold int) (subject, body string) {
	subject = fmt.Sprintf("【在庫不足】%s (SKU: %s)", productName, skuCode)
	data := map[string]any{
		"SKUCode":      skuCode,
		"ProductName":  productName,
		"CurrentStock": currentStock,
		"Threshold":    threshold,
	}
	body = render(lowStockAlertTmpl, data)
	return subject, body
}

func render(tmpl *template.Template, data any) string {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Sprintf("template error: %v", err)
	}
	return buf.String()
}
