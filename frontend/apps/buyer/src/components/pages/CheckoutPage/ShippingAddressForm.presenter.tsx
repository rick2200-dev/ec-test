/**
 * Plain shipping-address fields. Matches the loose shape forwarded as
 * raw JSON to the order service — tighten when the order service starts
 * validating canonical fields.
 */
export interface ShippingAddressFields {
  recipient: string;
  postal_code: string;
  region: string;
  city: string;
  line1: string;
  line2: string;
  phone: string;
}

export interface ShippingAddressFormPresenterProps {
  values: ShippingAddressFields;
  labels: {
    sectionTitle: string;
    recipient: string;
    postalCode: string;
    region: string;
    city: string;
    line1: string;
    line2: string;
    phone: string;
  };
  disabled?: boolean;
  onChange: (field: keyof ShippingAddressFields, value: string) => void;
}

export function ShippingAddressFormPresenter({
  values,
  labels,
  disabled,
  onChange,
}: ShippingAddressFormPresenterProps) {
  const field = (
    name: keyof ShippingAddressFields,
    label: string,
    type: "text" | "tel" = "text",
    required = false
  ) => (
    <div>
      <label htmlFor={`ship-${name}`} className="block text-sm font-medium text-gray-700">
        {label}
        {required && <span className="ml-1 text-red-500">*</span>}
      </label>
      <input
        id={`ship-${name}`}
        type={type}
        value={values[name]}
        onChange={(e) => onChange(name, e.target.value)}
        disabled={disabled}
        required={required}
        aria-required={required}
        className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 disabled:bg-gray-50"
      />
    </div>
  );

  return (
    <section className="rounded-lg border border-gray-200 bg-white p-6">
      <h2 className="text-base font-semibold text-gray-900">{labels.sectionTitle}</h2>
      <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
        {field("recipient", labels.recipient, "text", true)}
        {field("phone", labels.phone, "tel")}
        {field("postal_code", labels.postalCode, "text", true)}
        {field("region", labels.region, "text", true)}
        {field("city", labels.city, "text", true)}
        {field("line1", labels.line1, "text", true)}
        <div className="sm:col-span-2">{field("line2", labels.line2)}</div>
      </div>
    </section>
  );
}
