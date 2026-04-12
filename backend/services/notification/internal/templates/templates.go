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

var inquiryNewMessageTmpl = template.Must(template.New("inquiry_new_message").Parse(`
<!DOCTYPE html>
<html>
<body>
<h2>新着メッセージのお知らせ</h2>
<p>{{.RecipientLabel}} 様</p>
<p>{{.SenderLabel}} から新着メッセージがあります。</p>
<table>
  <tr><td><strong>件名:</strong></td><td>{{.Subject}}</td></tr>
  <tr><td><strong>商品:</strong></td><td>{{.ProductName}}</td></tr>
</table>
<blockquote>{{.BodyPreview}}</blockquote>
<p>詳細を確認するには、お問い合わせスレッドを開いてください。</p>
</body>
</html>
`))

var orderCancellationRequestedTmpl = template.Must(template.New("order_cancellation_requested").Parse(`
<!DOCTYPE html>
<html>
<body>
<h2>注文キャンセル申請を受け付けました</h2>
<p>出品者様</p>
<p>以下の注文に対してキャンセル申請が届いています。買い手側の理由をご確認の上、承認または却下してください。</p>
<table>
  <tr><td><strong>注文番号:</strong></td><td>{{.OrderID}}</td></tr>
  <tr><td><strong>申請理由:</strong></td><td>{{.Reason}}</td></tr>
</table>
<p>セラーダッシュボードの「キャンセル申請」画面から対応をお願いします。</p>
</body>
</html>
`))

var orderCancellationApprovedTmpl = template.Must(template.New("order_cancellation_approved").Parse(`
<!DOCTYPE html>
<html>
<body>
<h2>キャンセル申請が承認されました</h2>
<p>お客様</p>
<p>ご申請いただいた注文のキャンセルが承認され、返金処理が完了しました。</p>
<table>
  <tr><td><strong>注文番号:</strong></td><td>{{.OrderID}}</td></tr>
  <tr><td><strong>返金金額:</strong></td><td>¥{{.RefundAmount}}</td></tr>
</table>
<p>返金はご利用のクレジットカード会社を通じて数営業日以内に反映されます。</p>
</body>
</html>
`))

var orderCancellationRejectedTmpl = template.Must(template.New("order_cancellation_rejected").Parse(`
<!DOCTYPE html>
<html>
<body>
<h2>キャンセル申請について</h2>
<p>お客様</p>
<p>恐れ入りますが、ご申請いただいた注文のキャンセルは出品者により却下されました。</p>
<table>
  <tr><td><strong>注文番号:</strong></td><td>{{.OrderID}}</td></tr>
  <tr><td><strong>出品者からのコメント:</strong></td><td>{{.SellerComment}}</td></tr>
</table>
<p>ご不明な点があれば、出品者へ直接お問い合わせください。</p>
</body>
</html>
`))

var orderCancelledTmpl = template.Must(template.New("order_cancelled").Parse(`
<!DOCTYPE html>
<html>
<body>
<h2>注文がキャンセルされました</h2>
<p>お客様</p>
<p>以下の注文がキャンセルされました。</p>
<table>
  <tr><td><strong>注文番号:</strong></td><td>{{.OrderID}}</td></tr>
  <tr><td><strong>キャンセル理由:</strong></td><td>{{.Reason}}</td></tr>
</table>
<p>ご利用ありがとうございました。</p>
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

// InquiryNewMessageNotification generates an email notifying the recipient
// that a new message has been posted on a buyer↔seller inquiry thread.
func InquiryNewMessageNotification(recipientLabel, senderLabel, subjectText, productName, bodyPreview string) (subject, body string) {
	subject = fmt.Sprintf("【お問い合わせ】新着メッセージ: %s", subjectText)
	data := map[string]any{
		"RecipientLabel": recipientLabel,
		"SenderLabel":    senderLabel,
		"Subject":        subjectText,
		"ProductName":    productName,
		"BodyPreview":    bodyPreview,
	}
	body = render(inquiryNewMessageTmpl, data)
	return subject, body
}

// OrderCancellationRequested generates a seller email notifying them
// that a buyer has opened a cancellation request against one of their
// orders.
func OrderCancellationRequested(orderID, reason string) (subject, body string) {
	subject = fmt.Sprintf("【キャンセル申請】注文番号: %s", orderID)
	data := map[string]any{
		"OrderID": orderID,
		"Reason":  reason,
	}
	body = render(orderCancellationRequestedTmpl, data)
	return subject, body
}

// OrderCancellationApproved generates a buyer email confirming a
// cancellation has been approved and the refund has been issued.
func OrderCancellationApproved(orderID string, refundAmount int64) (subject, body string) {
	subject = fmt.Sprintf("【キャンセル承認】注文番号: %s", orderID)
	data := map[string]any{
		"OrderID":      orderID,
		"RefundAmount": refundAmount,
	}
	body = render(orderCancellationApprovedTmpl, data)
	return subject, body
}

// OrderCancellationRejected generates a buyer email telling them the
// seller rejected their cancellation request, with the seller's comment.
func OrderCancellationRejected(orderID, sellerComment string) (subject, body string) {
	subject = fmt.Sprintf("【キャンセル却下】注文番号: %s", orderID)
	data := map[string]any{
		"OrderID":       orderID,
		"SellerComment": sellerComment,
	}
	body = render(orderCancellationRejectedTmpl, data)
	return subject, body
}

// OrderCancelledNotification generates a buyer email as the final
// confirmation that their order has been cancelled. Fired off after
// the approval workflow finishes all of its side effects (refund,
// transfer reversals, stock release).
func OrderCancelledNotification(orderID, reason string) (subject, body string) {
	subject = fmt.Sprintf("【キャンセル完了】注文番号: %s", orderID)
	data := map[string]any{
		"OrderID": orderID,
		"Reason":  reason,
	}
	body = render(orderCancelledTmpl, data)
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

// ReviewCreatedNotification generates a seller email notifying them
// that a buyer posted a new review on one of their products.
func ReviewCreatedNotification(productName string, rating int, title string) (subject, body string) {
	subject = fmt.Sprintf("【レビュー】新着レビュー: %s", productName)
	data := map[string]any{
		"ProductName": productName,
		"Rating":      rating,
		"Title":       title,
	}
	body = render(reviewCreatedTmpl, data)
	return subject, body
}

// ReviewRepliedNotification generates a buyer email notifying them
// that a seller has replied to their review.
func ReviewRepliedNotification(productName, replyPreview string) (subject, body string) {
	subject = fmt.Sprintf("【レビュー返信】%s", productName)
	data := map[string]any{
		"ProductName":  productName,
		"ReplyPreview": replyPreview,
	}
	body = render(reviewRepliedTmpl, data)
	return subject, body
}

var reviewCreatedTmpl = template.Must(template.New("review_created").Parse(`
<!DOCTYPE html>
<html>
<body>
<h2>新着レビューのお知らせ</h2>
<p>出品者 様</p>
<p>あなたの商品にレビューが投稿されました。</p>
<table>
  <tr><td><strong>商品名:</strong></td><td>{{.ProductName}}</td></tr>
  <tr><td><strong>評価:</strong></td><td>{{.Rating}} / 5</td></tr>
  <tr><td><strong>タイトル:</strong></td><td>{{.Title}}</td></tr>
</table>
<p>詳細はセラーダッシュボードからご確認いただけます。</p>
</body>
</html>
`))

var reviewRepliedTmpl = template.Must(template.New("review_replied").Parse(`
<!DOCTYPE html>
<html>
<body>
<h2>レビュー返信のお知らせ</h2>
<p>購入者 様</p>
<p>あなたのレビューに出品者から返信がありました。</p>
<table>
  <tr><td><strong>商品名:</strong></td><td>{{.ProductName}}</td></tr>
  <tr><td><strong>返信内容:</strong></td><td>{{.ReplyPreview}}</td></tr>
</table>
<p>詳細は商品ページからご確認いただけます。</p>
</body>
</html>
`))

func render(tmpl *template.Template, data any) string {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Sprintf("template error: %v", err)
	}
	return buf.String()
}
