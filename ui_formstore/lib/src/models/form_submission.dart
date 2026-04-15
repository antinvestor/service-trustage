/// Status of a form submission.
enum SubmissionStatus {
  pending,
  complete,
  archived;

  static SubmissionStatus fromString(String value) {
    return SubmissionStatus.values.firstWhere(
      (s) => s.name == value,
      orElse: () => SubmissionStatus.pending,
    );
  }
}

/// A form submission with its data payload.
class FormSubmission {
  const FormSubmission({
    required this.id,
    required this.formId,
    this.submitterId = '',
    this.status = SubmissionStatus.pending,
    this.data = const {},
    this.fileCount = 0,
    this.idempotencyKey = '',
    this.metadata,
    this.createdAt,
    this.modifiedAt,
  });

  final String id;
  final String formId;
  final String submitterId;
  final SubmissionStatus status;
  final Map<String, dynamic> data;
  final int fileCount;
  final String idempotencyKey;
  final Map<String, dynamic>? metadata;
  final DateTime? createdAt;
  final DateTime? modifiedAt;

  factory FormSubmission.fromJson(Map<String, dynamic> json) {
    return FormSubmission(
      id: json['id'] as String? ?? '',
      formId: json['form_id'] as String? ?? '',
      submitterId: json['submitter_id'] as String? ?? '',
      status: SubmissionStatus.fromString(json['status'] as String? ?? ''),
      data: (json['data'] as Map<String, dynamic>?) ?? const {},
      fileCount: json['file_count'] as int? ?? 0,
      idempotencyKey: json['idempotency_key'] as String? ?? '',
      metadata: json['metadata'] as Map<String, dynamic>?,
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
      if (submitterId.isNotEmpty) 'submitter_id': submitterId,
      'data': data,
      if (idempotencyKey.isNotEmpty) 'idempotency_key': idempotencyKey,
      if (metadata != null) 'metadata': metadata,
    };
  }
}
