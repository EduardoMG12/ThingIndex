package crypto

import "testing"

const masterKey = "chave-mestra-de-teste-nao-usar-em-producao"

func TestEncryptDecryptRoundTrip(t *testing.T) {
	t.Parallel()

	secret := "sk-proj-credencial-do-provider-do-usuario"

	encrypted, err := Encrypt(secret, masterKey, 42)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if encrypted == secret {
		t.Fatal("texto cifrado igual ao original")
	}

	decrypted, err := Decrypt(encrypted, masterKey, 42)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if decrypted != secret {
		t.Fatalf("round trip devolveu %q, quero %q", decrypted, secret)
	}
}

// A chave e derivada com o userID no salt. Uma credencial vazada do usuario A
// nao pode ser aberta no contexto do usuario B, mesmo com a mesma chave mestra.
func TestDecryptRejectsOtherUser(t *testing.T) {
	t.Parallel()

	encrypted, err := Encrypt("credencial do usuario 1", masterKey, 1)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	if _, err := Decrypt(encrypted, masterKey, 2); err == nil {
		t.Fatal("decifrou credencial de outro usuario")
	}
}

func TestDecryptRejectsWrongMasterKey(t *testing.T) {
	t.Parallel()

	encrypted, err := Encrypt("credencial", masterKey, 7)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	if _, err := Decrypt(encrypted, "chave-mestra-errada", 7); err == nil {
		t.Fatal("decifrou com chave mestra errada")
	}
}

// GCM autentica o texto cifrado: adulterar um byte tem de falhar, nao devolver
// texto claro corrompido.
func TestDecryptRejectsTamperedCiphertext(t *testing.T) {
	t.Parallel()

	encrypted, err := Encrypt("credencial", masterKey, 7)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	tampered := []byte(encrypted)
	// Mexe num caractere do meio, mantendo o base64 valido.
	middle := len(tampered) / 2
	if tampered[middle] == 'A' {
		tampered[middle] = 'B'
	} else {
		tampered[middle] = 'A'
	}

	if _, err := Decrypt(string(tampered), masterKey, 7); err == nil {
		t.Fatal("aceitou texto cifrado adulterado")
	}
}

func TestDecryptRejectsShortCiphertext(t *testing.T) {
	t.Parallel()

	if _, err := Decrypt("YWJj", masterKey, 1); err == nil {
		t.Fatal("aceitou texto cifrado menor que o nonce")
	}
}

// Nonce aleatorio por chamada: cifrar o mesmo valor duas vezes nao pode produzir
// a mesma saida, senao da para inferir credenciais iguais entre usuarios.
func TestEncryptUsesFreshNonce(t *testing.T) {
	t.Parallel()

	first, err := Encrypt("mesmo segredo", masterKey, 1)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	second, err := Encrypt("mesmo segredo", masterKey, 1)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	if first == second {
		t.Fatal("duas cifragens do mesmo valor produziram a mesma saida")
	}
}
