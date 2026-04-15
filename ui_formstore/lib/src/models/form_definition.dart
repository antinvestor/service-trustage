/// A form definition with its JSON schema and metadata.
class FormDefinition {
  const FormDefinition({
    required this.id,
    required this.formId,
    required this.name,
    this.description = '',
    this.active = true,
    this.jsonSchema,
    this.createdAt,
    this.modifiedAt,
  });

  final String id;
  final String formId;
  final String name;
  final String description;
  final bool active;
  final Map<String, dynamic>? jsonSchema;
  final DateTime? createdAt;
  final DateTime? modifiedAt;

  factory FormDefinition.fromJson(Map<String, dynamic> json) {
    return FormDefinition(
      id: json['id'] as String? ?? '',
      formId: json['form_id'] as String? ?? '',
      name: json['name'] as String? ?? '',
      description: json['description'] as String? ?? '',
      active: json['active'] as bool? ?? true,
      jsonSchema: json['json_schema'] as Map<String, dynamic>?,
      createdAt: json['created_at'] != null
          ? DateTime.tryParse(json['created_at'] as String)
          : null,
      modifiedAt: json['modified_at'] != null
          ? DateTime.tryParse(json['modified_at'] as String)
          : null,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'form_id': formId,
      'name': name,
      'description': description,
      'active': active,
      if (jsonSchema != null) 'json_schema': jsonSchema,
    };
  }

  FormDefinition copyWith({
    String? name,
    String? description,
    bool? active,
    Map<String, dynamic>? jsonSchema,
  }) {
    return FormDefinition(
      id: id,
      formId: formId,
      name: name ?? this.name,
      description: description ?? this.description,
      active: active ?? this.active,
      jsonSchema: jsonSchema ?? this.jsonSchema,
      createdAt: createdAt,
      modifiedAt: modifiedAt,
    );
  }
}
